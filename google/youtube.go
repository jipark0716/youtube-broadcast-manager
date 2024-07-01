package google

import (
	"fmt"
	"golang.org/x/oauth2"
	"google.golang.org/api/youtube/v3"
	"os/exec"
	"path/filepath"
	"rtmp/ffmpeg"
	"time"
)

var OauthConfig = &oauth2.Config{
	ClientID:     "216583567044-bqavjphj5v6dahtg1tolmmnk6nlaacae.apps.googleusercontent.com",
	ClientSecret: "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",
	RedirectURL:  "http://localhost:5862/oauth/google/redirect",
	Scopes:       []string{"email", "https://www.googleapis.com/auth/youtube"},
}

var BackUpIndex = 0
var ConvertIndex = 0
var StreamProcesses = make([]*Stream, 0)

type StreamStatus string

const (
	Ready   StreamStatus = "ready"
	Running StreamStatus = "running"
	Stopped StreamStatus = "stopped"
)

type Stream struct {
	ProfileId string
	Resource  *youtube.LiveBroadcast
	Process   *exec.Cmd
	Stream    *youtube.LiveStream
	Source    string
	Backup    int
}

type ProfileWithChannel struct {
	Channels []*youtube.Channel `json:"channels,omitempty"`
	Profile  *Profile           `json:"profile,omitempty"`
}

func GetAllChannels() (result []*ProfileWithChannel) {
	result = make([]*ProfileWithChannel, 0)
	for _, profile := range profiles {
		channelResponse, err := profile.GetChannels()

		if err == nil {
			result = append(result, &ProfileWithChannel{
				Channels: channelResponse.Items,
				Profile:  profile,
			})
		}
	}

	return
}

func ConvertCodec(source string) (path string, err error) {
	ConvertIndex += 1
	path, err = filepath.Abs(fmt.Sprintf("sources/%d.mkv", ConvertIndex))

	if err != nil {
		return
	}

	_, err = ffmpeg.ConvertCodec(source, path)

	return
}

func (p *Profile) FindOrCreateStream() (stream *youtube.LiveStream, err error) {
	streamResponse, err := p.GetStreams()

	if err != nil {
		return
	}

	index := 1
	for _, process := range StreamProcesses {
		if process.ProfileId == p.Id {
			index += 1
		}
	}

	if len(streamResponse.Items) > index {
		return streamResponse.Items[index], nil
	}

	return p.CreateStreams(index)
}

func CreateLive(profileId string) (broadCast *youtube.LiveBroadcast, err error) {
	profile := profiles[profileId]

	stream, err := profile.FindOrCreateStream()
	if err != nil {
		return
	}

	broadCast, err = profile.CreateBroadCast(stream)

	if err != nil {
		return
	}

	BackUpIndex += 1
	streamProcess := &Stream{
		ProfileId: profileId,
		Resource:  broadCast,
		Stream:    stream,
		Source:    "/Users/jipark/Workspace/ffmpeg test.mkv",
		Backup:    BackUpIndex,
	}
	streamProcess.Process, err = ffmpeg.StartStreaming(
		streamProcess.Source,
		streamProcess.Stream.Cdn.IngestionInfo.StreamName,
		streamProcess.Backup)
	if err != nil {
		return
	}

	StreamProcesses = append(StreamProcesses, streamProcess)

	return
}

func (p *Profile) GetChannels() (*youtube.ChannelListResponse, error) {
	return Retry(p, p.getChannels)
}

func (p *Profile) getChannels() (response *youtube.ChannelListResponse, err error) {
	service, err := p.GetYoutubeService()
	if err != nil {
		return
	}

	return service.
		Channels.
		List([]string{"id", "snippet"}).
		Mine(true).
		Do()
}

func (p *Profile) GetStreams() (*youtube.LiveStreamListResponse, error) {
	return Retry(p, p.getStreams)
}

func (p *Profile) getStreams() (response *youtube.LiveStreamListResponse, err error) {
	service, err := p.GetYoutubeService()
	if err != nil {
		return
	}

	return service.
		LiveStreams.
		List([]string{"id", "cdn"}).
		Mine(true).
		Do()
}

func (p *Profile) CreateStreams(index int) (*youtube.LiveStream, error) {
	return Retry(p, func() (*youtube.LiveStream, error) {
		return p.createStreams(index)
	})
}

func (p *Profile) createStreams(index int) (response *youtube.LiveStream, err error) {
	service, err := p.GetYoutubeService()
	if err != nil {
		return
	}

	return service.
		LiveStreams.
		Insert([]string{"snippet", "cdn"}, &youtube.LiveStream{
			Snippet: &youtube.LiveStreamSnippet{
				Title: fmt.Sprintf("제목-%d", index),
			},
			Cdn: &youtube.CdnSettings{
				IngestionType: "rtmp",
				FrameRate:     "variable",
				Resolution:    "variable",
			},
		}).
		Do()
}

func (p *Profile) StartBroadCast(id string) (*youtube.LiveBroadcast, error) {
	return Retry(p, func() (*youtube.LiveBroadcast, error) {
		return p.startBroadCast(id)
	})
}

func (p *Profile) startBroadCast(id string) (response *youtube.LiveBroadcast, err error) {
	service, err := p.GetYoutubeService()
	if err != nil {
		return
	}

	return service.
		LiveBroadcasts.
		Transition("live", id, []string{"id", "status", "snippet", "contentDetails"}).
		Do()
}

func (p *Profile) CreateBroadCast(stream *youtube.LiveStream) (*youtube.LiveBroadcast, error) {
	return Retry(p, func() (*youtube.LiveBroadcast, error) {
		return p.createBroadCast(stream)
	})
}

func (p *Profile) createBroadCast(stream *youtube.LiveStream) (response *youtube.LiveBroadcast, err error) {
	service, err := p.GetYoutubeService()
	if err != nil {
		return
	}

	response, err = service.
		LiveBroadcasts.
		Insert(
			[]string{"id", "status", "snippet", "contentDetails"},
			&youtube.LiveBroadcast{
				Status: &youtube.LiveBroadcastStatus{
					PrivacyStatus: "public",
					//SelfDeclaredMadeForKids: false,
				},
				Snippet: &youtube.LiveBroadcastSnippet{
					Title:              "제목요",
					Description:        "내용요",
					ScheduledStartTime: time.Now().Format("2006-01-02 15:04") + ":00",
				},
				ContentDetails: &youtube.LiveBroadcastContentDetails{
					EnableAutoStart: true,
					EnableAutoStop:  true,
				},
			},
		).
		Do()

	if err != nil {
		return
	}

	return service.
		LiveBroadcasts.
		Bind(response.Id, []string{"id", "status", "snippet", "contentDetails"}).
		StreamId(stream.Id).
		Do()
}
