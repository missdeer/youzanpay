package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	token YouzanToken
)

type YouzanToken struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	Timestamp   time.Time
}

func getToken() {
	resp, err := http.PostForm("https://open.youzan.com/oauth/token",
		url.Values{
			"client_id":     {clientID},
			"client_secret": {clientSecret},
			"grant_type":    {"silent"},
			"kdt_id":        {kdtID},
		})
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = json.Unmarshal(content, &token)
	if err != nil {
		fmt.Println(err)
		return
	}
	token.Timestamp = time.Now()
}

type CreateQRCodeResponse struct {
	Response struct {
		QRID   string `json:"qr_id"`
		QRURL  string `json:"qr_url"`
		QRCode string `json:"qr_code"`
		QRType int    `json:"qr_type"`
	} `json:"response"`
	ErrorResponse struct {
		Code    int    `json:"code"`
		Message string `json:"msg"`
	} `json:"error_response"`
}

type PushMessage struct {
	Mode      int    `json:"mode"`
	ID        string `json:"id"`
	ClientID  string `json:"client_id"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Message   string `json:"msg"`
	KdtID     int    `json:"kdt_id"`
	Sign      string `json:"sign"`
	Version   int    `json:"version"`
	Test      bool   `json:"test"`
	SendCount int    `json:"send_count"`
}

type DetailedTradeInfo struct {
	Response struct {
		Trade struct {
			QRID  string `json:"qr_id"`
			TID   string `json:"tid"`
			Price string `json:"price"`
		} `json:"trade"`
	} `json:"response"`
}

func getDetailedTradeInfo(tid string) (string, error) {
	c := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://open.youzan.com/api/oauthentry/youzan.trade/3.0.0/get", nil)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	q := req.URL.Query()
	q.Add("access_token", token.AccessToken)
	q.Add("tid", tid)
	req.URL.RawQuery = q.Encode()

	resp, err := c.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	fmt.Println(string(content))

	info := DetailedTradeInfo{}
	err = json.Unmarshal(content, &info)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return info.Response.Trade.QRID, nil
}

func callback(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "success",
	})
	b, e := c.GetRawData()
	if e != nil {
		fmt.Println(e)
		return
	}

	fmt.Println(string(b))
	pm := PushMessage{}
	e = json.Unmarshal(b, &pm)
	if e != nil {
		fmt.Println(e)
		return
	}

	if pm.Test == true {
		fmt.Println("ignore test message")
		return
	}

	if pm.Mode != 1 {
		fmt.Println("unexpected mode:", pm.Mode)
		return
	}

	if pm.ClientID != clientID || strconv.Itoa(pm.KdtID) != kdtID {
		fmt.Println("client id or KDT id not match")
		return
	}

	// get detailed info of trade by pm.ID
	qrID, err := getDetailedTradeInfo(pm.ID)
	if err != nil {
		return
	}

	handleWebscoketClient(qrID, &pm)
	handleTimerPollClient(qrID, &pm)
}
