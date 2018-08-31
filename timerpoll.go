package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	qrTradeMap     = sync.Map{} // qr id, trade id
	tradeStatusMap = sync.Map{} // trade id, status
)

func handleTimerPollClient(qrID string, pm *PushMessage) {
	qrTradeMap.Store(qrID, pm.ID)
	tradeStatusMap.Store(pm.ID, pm.Status)
	if pm.Status == "TRADE_SUCCESS" ||
		pm.Status == "TRADE_CLOSED" {
		clientQRMap.Delete(qrID)
	}
}

func createQRCode(c *gin.Context) {
	description := c.PostForm("title")
	price := c.PostForm("price")

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://open.youzan.com/api/oauthentry/youzan.pay.qrcode/3.0.0/create", nil)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  err,
		})
		return
	}

	q := req.URL.Query()
	q.Add("access_token", token.AccessToken)
	q.Add("qr_name", description)
	q.Add("qr_price", price)
	q.Add("qr_type", "QR_TYPE_DYNAMIC")
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  err,
		})
		return
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  err,
		})
		return
	}

	//fmt.Println(string(content))

	r := CreateQRCodeResponse{}
	if err = json.Unmarshal(content, &r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  err,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id": r.Response.QRID,
		"qr": r.Response.QRCode,
	})
}

func queryOrderStatus(c *gin.Context) {
	qrID := c.PostForm("id")

	v, ok := qrTradeMap.Load(qrID)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"msg":    "qtcode-created",
		})
		return
	}

	tradeID, ok := v.(string)
	if !ok {
		c.JSON(http.StatusOK, gin.H{
			"status": "error",
			"error":  "invalid trade ID",
		})
		return
	}

	v, ok = tradeStatusMap.Load(tradeID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "trade not found",
		})
		return
	}

	status, ok := v.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"status": "error",
			"error":  "invalid trade status",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_state": status,
	})
}
