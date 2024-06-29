package google

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

const (
	OauthEndpoint = "https://accounts.google.com/o/oauth2/v2/auth"
	RedirectUrl   = "http://localhost:5862/oauth/google/redirect"
	ClientId      = "216583567044-bqavjphj5v6dahtg1tolmmnk6nlaacae.apps.googleusercontent.com"
	ClientSecret  = "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"
	Scopes        = "email https://www.googleapis.com/auth/youtube"
)

func Route(router *gin.Engine) {
	router.GET("/oauth/google", Oauth)
	router.GET("/oauth/google/redirect", Redirect)
	router.GET("/channels", Channels)
	router.POST("/live-start", StartLive)
}

func Oauth(c *gin.Context) {
	c.Redirect(
		http.StatusFound,
		fmt.Sprintf(
			"%s?client_id=%s&redirect_uri=%s&response_type=code&scope=%s&access_type=offline",
			OauthEndpoint,
			ClientId,
			RedirectUrl,
			Scopes),
	)
}

func Redirect(c *gin.Context) {
	c.Redirect(http.StatusFound, "/")
	code := c.Query("code")
	if code == "" {
		return
	}
	token, err := NewAccountFromAuthCode(code)
	if err != nil {
		return
	}
	_ = SaveToken(token)
}

func Channels(c *gin.Context) {
	c.JSON(http.StatusOK, GetAllChannels())
}

func StartLive(c *gin.Context) {
	profileId := ""
	var has bool
	if profileId, has = c.GetPostForm("profileId"); !has {
		c.JSON(http.StatusBadRequest, gin.H{
			"code": http.StatusBadRequest,
		})
		return
	}

	live, err := CreateLive(profileId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": fmt.Sprintf("%#v", err),
		})
		return
	}

	c.JSON(http.StatusOK, live)
}
