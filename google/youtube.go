package google

import (
	"fmt"
	"golang.org/x/oauth2"
	"google.golang.org/api/youtube/v3"
	"os"
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
var ConvertIndex = time.Now().Unix()
var StreamProcesses = make(map[string]*Stream)

type Stream struct {
	ProfileId string                 `json:"profile_id,omitempty"`
	Resource  *youtube.LiveBroadcast `json:"resource,omitempty"`
	Process   *exec.Cmd              `json:"-"`
	Stream    *youtube.LiveStream    `json:"stream,omitempty"`
	Source    string                 `json:"source,omitempty"`
	Backup    int                    `json:"backup,omitempty"`
}

func init() {
	location, err := time.LoadLocation("UTC")
	if err != nil {
		panic(err)
	}

	time.Local = location

	go func() {
		for {
			for i, process := range StreamProcesses {
				profile := profiles[process.ProfileId]
				broadcast, err := profile.GetBroadcast(i)

				if err != nil {
					_ = StopStream(i)
					continue
				}

				stopStatus := true
				for _, status := range []string{"live", "liveStarting", "ready", "testStarting"} {
					if status == broadcast.Status.LifeCycleStatus {
						stopStatus = false
						break
					}
				}
				if stopStatus {
					_ = StopStream(i)
				}
			}
		}
	}()
}

func StopStream(streamId string) error {
	stream := StreamProcesses[streamId]
	delete(StreamProcesses, streamId)

	return stream.Process.Process.Kill()
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

func CreateLive(profileId, title, description, categoryId, thumbnail, source string) (broadCast *youtube.LiveBroadcast, err error) {
	profile := profiles[profileId]

	stream, err := profile.FindOrCreateStream()
	if err != nil {
		return
	}

	broadCast, err = profile.CreateBroadCast(stream, title, description)

	if err != nil {
		return
	}

	_, err = profile.SetThumbnail(broadCast, thumbnail)

	if err != nil {
		return
	}

	_, err = profile.ChangeVideoCategory(broadCast, categoryId)

	if err != nil {
		return
	}

	broadCast.Snippet.Thumbnails.High.Url = thumbnail

	BackUpIndex += 1
	streamProcess := &Stream{
		ProfileId: profileId,
		Resource:  broadCast,
		Stream:    stream,
		Source:    source,
		Backup:    BackUpIndex,
	}
	streamProcess.Process, err = ffmpeg.StartStreaming(
		streamProcess.Source,
		streamProcess.Stream.Cdn.IngestionInfo.StreamName,
		streamProcess.Backup)
	if err != nil {
		return
	}

	StreamProcesses[streamProcess.Resource.Id] = streamProcess

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

func GetVideoCategories() (*youtube.VideoCategoryListResponse, error) {
	for _, profile := range profiles {
		return profile.GetCategoryList()
	}
	return nil, fmt.Errorf("token not found")
}

func (p *Profile) SetThumbnail(broadcast *youtube.LiveBroadcast, thumbnail string) (*youtube.ThumbnailSetResponse, error) {
	return Retry(p, func() (*youtube.ThumbnailSetResponse, error) {
		return p.setThumbnail(broadcast, thumbnail)
	})
}

func (p *Profile) setThumbnail(broadcast *youtube.LiveBroadcast, thumbnail string) (response *youtube.ThumbnailSetResponse, err error) {
	service, err := p.GetYoutubeService()
	if err != nil {
		return
	}

	reader, err := os.Open(thumbnail)
	if err != nil {
		return
	}

	return service.
		Thumbnails.
		Set(broadcast.Id).
		Media(reader).
		Do()
}

func (p *Profile) GetBroadcast(id string) (*youtube.LiveBroadcast, error) {
	return Retry(p, func() (*youtube.LiveBroadcast, error) {
		return p.getBroadcast(id)
	})
}

func (p *Profile) getBroadcast(id string) (response *youtube.LiveBroadcast, err error) {
	service, err := p.GetYoutubeService()
	if err != nil {
		return
	}

	res, err := service.
		LiveBroadcasts.
		List([]string{}).
		Id(id).
		Do()

	if err != nil {
		return
	}

	if len(res.Items) == 0 {
		err = fmt.Errorf("not found")
		return
	}

	return res.Items[0], nil
}

func (p *Profile) CreateBroadCast(stream *youtube.LiveStream, title, description string) (*youtube.LiveBroadcast, error) {
	return Retry(p, func() (*youtube.LiveBroadcast, error) {
		return p.createBroadCast(stream, title, description)
	})
}

func (p *Profile) createBroadCast(stream *youtube.LiveStream, title, description string) (response *youtube.LiveBroadcast, err error) {
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
				},
				Snippet: &youtube.LiveBroadcastSnippet{
					Title:              title,
					Description:        description,
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

func (p *Profile) GetCategoryList() (*youtube.VideoCategoryListResponse, error) {
	return Retry(p, p.getCategoryList)
}

func (p *Profile) getCategoryList() (response *youtube.VideoCategoryListResponse, err error) {
	service, err := p.GetYoutubeService()
	if err != nil {
		return
	}

	return service.
		VideoCategories.
		List([]string{"snippet", "id"}).
		RegionCode("KR").
		Hl("ko_KR").
		Do()
}

func (p *Profile) ChangeVideoCategory(broadcast *youtube.LiveBroadcast, categoryId string) (*youtube.Video, error) {
	return Retry(p, func() (*youtube.Video, error) {
		return p.changeVideoCategory(broadcast, categoryId)
	})
}

func (p *Profile) changeVideoCategory(broadcast *youtube.LiveBroadcast, categoryId string) (response *youtube.Video, err error) {
	service, err := p.GetYoutubeService()
	if err != nil {
		return
	}

	return service.
		Videos.
		Update([]string{"snippet"}, &youtube.Video{
			Id: broadcast.Id,
			Snippet: &youtube.VideoSnippet{
				Title:      broadcast.Snippet.Title,
				CategoryId: categoryId,
			},
		}).
		Do()
}
