package main

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	clientID     string
	clientSecret string
	kdtID        string
)

func main() {
	if v, ok := os.LookupEnv("YOUZANPAY_CLIENTID"); ok {
		clientID = v
	}
	if v, ok := os.LookupEnv("YOUZANPAY_CLIENTSECRET"); ok {
		clientSecret = v
	}
	if v, ok := os.LookupEnv("YOUZANPAY_KDTID"); ok {
		kdtID = v
	}
	getToken()
	go func() {
		for {
			time.Sleep(time.Second * time.Duration(token.ExpiresIn-600))
			getToken()
		}
	}()

	addr := ":8089"
	if bind, ok := os.LookupEnv("YOUZANPAY_BIND"); ok {
		addr = bind
	}
	r := gin.Default()
	r.LoadHTMLGlob("templates/*")
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/d")
	})
	r.GET("/ws", wsHandler)
	r.GET("/w", func(c *gin.Context) {
		c.HTML(http.StatusOK, "websocket.tmpl", gin.H{
			"title": "个人网站在线收款解决方案演示websocket版",
		})
	})
	r.GET("/d", func(c *gin.Context) {
		c.HTML(http.StatusOK, "timerpoll.tmpl", gin.H{
			"title": "个人网站在线收款解决方案演示timer poll版",
		})
	})
	r.POST("/create/qrcode", createQRCode)
	r.POST("/query/orderstatus", queryOrderStatus)
	r.POST("/callback", callback)
	r.Run(addr)
}
