package google

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
	"io"
	"net/http"
	"net/url"
	"os"
)

type Profile struct {
	Id            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Picture       string `json:"picture"`
	Token         *Token
}

type Token struct {
	oauth2.Token

	RefreshToken string `json:"refresh_token,omitempty"`
}

const TokenFile = "./token.json"

var profiles map[string]*Profile

func init() {
	profiles = map[string]*Profile{}

	fileContent, err := os.ReadFile(TokenFile)
	if err != nil {
		return
	}

	json.Unmarshal(fileContent, &profiles)
}

func saveProfile() {
	payload, err := json.Marshal(profiles)
	if err != nil {
		return
	}
	_ = os.WriteFile(TokenFile, payload, os.ModePerm)
}

func Retry[T interface{}](p *Profile, action func() (T, error)) (response T, err error) {
	for i := 1; i <= 3; i++ {
		response, err = action()
		if err == nil {
			return
		}
		if p.Token.RefreshToken == "" {
			return
		}
		token, err := TokenRefresh(p.Token.RefreshToken)
		if err == nil {
			p.Token = token
			saveProfile()
		} else {
			delete(profiles, p.Id)
		}
	}

	return
}

func (p *Profile) GetYoutubeService() (youtubeService *youtube.Service, err error) {
	background := context.Background()
	youtubeService, err = youtube.NewService(
		background,
		option.WithTokenSource(OauthConfig.TokenSource(background, &p.Token.Token)))

	if err != nil {
		delete(profiles, p.Id)
		saveProfile()
	}

	return
}

func TokenRefresh(refreshToken string) (token *Token, err error) {
	response, err := http.PostForm("https://oauth2.googleapis.com/token", url.Values{
		"client_id":     {ClientId},
		"client_secret": {ClientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
	})
	if err != nil {
		return
	}
	defer response.Body.Close()

	responseBuffer, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf(fmt.Sprintf("fail refresh %d %s", response.StatusCode, string(responseBuffer)))
	}

	err = json.Unmarshal(responseBuffer, &token)

	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}
	return
}

func NewAccountFromAuthCode(code string) (token *Token, err error) {
	response, err := http.PostForm("https://oauth2.googleapis.com/token", url.Values{
		"client_id":     {ClientId},
		"client_secret": {ClientSecret},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {RedirectUrl},
		"code":          {code},
	})
	if err != nil {
		return
	}
	defer response.Body.Close()

	responseBuffer, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(responseBuffer, &token)
	return
}

func SaveToken(token *Token) (err error) {
	response, err := http.Get(fmt.Sprintf("https://www.googleapis.com/oauth2/v2/userinfo?access_token=%s", token.AccessToken))
	if err != nil {
		return
	}
	defer response.Body.Close()

	var profile *Profile
	responseBuffer, err := io.ReadAll(response.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(responseBuffer, &profile)
	profile.Token = token
	profiles[profile.Id] = profile
	saveProfile()
	return
}
