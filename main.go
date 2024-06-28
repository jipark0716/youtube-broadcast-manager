package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"rtmp/google"
)

const PORT = 5862

func main() {
	router := gin.Default()
	router.LoadHTMLGlob("resource/*.html")
	google.Route(router)
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	go func() {
		//browser.OpenURL(fmt.Sprintf("http://localhost:%d", PORT))
	}()

	router.Run(fmt.Sprintf(":%d", PORT))

	select {}
}
