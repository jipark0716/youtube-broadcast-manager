package google

import (
	"golang.org/x/oauth2"
	"google.golang.org/api/youtube/v3"
	"os/exec"
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
var StreamProcesses = make([]*Stream, 0)

type Stream struct {
	Resource *youtube.LiveBroadcast
	Process  *exec.Cmd
	Stream   *youtube.LiveStream
	Source   string
	Backup   int
}

func GetAllChannels() (channels []*youtube.Channel) {
	channels = make([]*youtube.Channel, 0)

	for _, profile := range profiles {
		channelResponse, err := profile.GetChannels()

		if err == nil {
			channels = append(channels, channelResponse.Items...)
		}
	}

	return
}

func CreateLive(profileId string) (broadCast *youtube.LiveBroadcast, err error) {
	profile := profiles[profileId]

	streamResponse, err := profile.GetStreams()

	if err != nil {
		return
	}

	stream := streamResponse.Items[0]
	broadCast, err = profile.CreateBroadCast(stream)

	if err != nil {
		return
	}

	BackUpIndex += 1
	streamProcess := &Stream{
		Resource: broadCast,
		Stream:   stream,
		Source:   "/Users/jipark/Workspace/ffmpeg test.mp4",
		Backup:   BackUpIndex,
	}
	streamProcess.Process, err = ffmpeg.StartStreaming(
		streamProcess.Source,
		streamProcess.Stream.Cdn.IngestionInfo.StreamName,
		streamProcess.Backup)
	if err != nil {
		return
	}

	StreamProcesses = append(StreamProcesses, streamProcess)
	println("pre polling")
	time.Sleep(time.Second * 6)
	started := false
	for i := 0; i < 40; i++ {
		println("polling")
		_, err := profile.StartBroadCast(streamProcess.Resource.Id)
		if err == nil {
			started = true
			break
		}
		time.Sleep(time.Second * 2)
	}

	// 시작 실패시 리소스 삭제
	if !started {

	}

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

//func (p *Profile) CreateStreams() (*youtube.LiveStream, error) {
//	return Retry(p, func() (*youtube.LiveStream, error) {
//		return p.createStreams()
//	})
//}
//
//func (p *Profile) createStreams() (response *youtube.LiveStream, err error) {
//	service, err := p.GetYoutubeService()
//	if err != nil {
//		return
//	}
//
//	return service.
//		LiveStreams.
//		Insert().
//		Do()
//}

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

	return service.
		LiveBroadcasts.
		Insert(
			[]string{"id", "status", "snippet", "contentDetails"},
			&youtube.LiveBroadcast{
				Status: &youtube.LiveBroadcastStatus{
					PrivacyStatus: "public",
				},
				Snippet: &youtube.LiveBroadcastSnippet{
					Title:              "제목요",
					Description:        "내용요",
					ScheduledStartTime: "2024-06-26 20:00:00",
				},
				ContentDetails: &youtube.LiveBroadcastContentDetails{
					BoundStreamId: stream.Cdn.IngestionInfo.StreamName,
				},
			},
		).
		Do()
}
