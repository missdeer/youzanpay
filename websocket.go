package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	upGrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	clientQRMap     = sync.Map{} // qr id, client id
	clientWSConnMap = sync.Map{} // client id, websocket connection
)

type WSResponse struct {
	Code    int    `json:"code"`
	Event   string `json:"event"`
	Data    string `json:"data"`
	QRURL   string `json:"qr_url"`
	QRCode  string `json:"qr_code"`
	Message string `json:"msg"`
}

//webSocket请求ping 返回pong
func wsHandler(c *gin.Context) {
	//升级get请求为webSocket协议
	ws, err := upGrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()
	var wsClientID string
	for {
		//读取ws中的数据
		mt, message, err := ws.ReadMessage()
		if err != nil {
			fmt.Println("read from ws failed", err)
			break
		}
		m := string(message)
		ms := strings.Split(m, ",")
		if len(ms) < 3 {
			fmt.Println("invalid parameters")
			break
		}
		wsClientID = ms[0]
		clientWSConnMap.Store(wsClientID, ws)
		_, err = strconv.Atoi(ms[1])
		if err != nil {
			fmt.Println("invalid price", err)
			break
		}
		description := ms[2]

		c := &http.Client{
			Timeout: 15 * time.Second,
		}

		req, err := http.NewRequest("GET", "https://open.youzan.com/api/oauthentry/youzan.pay.qrcode/3.0.0/create", nil)
		if err != nil {
			fmt.Println(err)
			break
		}

		q := req.URL.Query()
		q.Add("access_token", token.AccessToken)
		q.Add("qr_name", description)
		q.Add("qr_price", ms[1])
		q.Add("qr_type", "QR_TYPE_DYNAMIC")
		req.URL.RawQuery = q.Encode()

		resp, err := c.Do(req)
		if err != nil {
			fmt.Println(err)
			break
		}
		defer resp.Body.Close()

		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			break
		}

		//fmt.Println(string(content))

		r := CreateQRCodeResponse{}
		if err = json.Unmarshal(content, &r); err != nil {
			fmt.Println(err)
			break
		}

		wsr := WSResponse{
			Code:    200,
			Event:   "create",
			QRURL:   r.Response.QRURL,
			QRCode:  r.Response.QRCode,
			Message: r.ErrorResponse.Message,
		}

		if r.ErrorResponse.Code != 0 {
			wsr.Code = r.ErrorResponse.Code
		}

		b, err := json.Marshal(&wsr)
		if err != nil {
			fmt.Println(err)
			break
		}

		clientQRMap.Store(r.Response.QRID, wsClientID)
		//写入ws数据
		err = ws.WriteMessage(mt, b)
		if err != nil {
			fmt.Println(err)
			break
		}
	}
	clientWSConnMap.Delete(wsClientID)
}

func handleWebscoketClient(qrID string, pm *PushMessage) {
	v, ok := clientQRMap.Load(qrID)
	if !ok {
		fmt.Println("qr id not found")
		return
	}
	wsClientID, ok := v.(string)
	conn, ok := clientWSConnMap.Load(wsClientID)
	if !ok {
		fmt.Println("connection not found")
		return
	}
	ws, ok := conn.(*websocket.Conn)
	if !ok {
		fmt.Println("ws connection not found")
		return
	}
	if pm.Type == "TRADE_ORDER_STATE" {
		if pm.Status == "WAIT_BUYER_PAY" {
			ws.WriteMessage(websocket.TextMessage, []byte(`{"code":200,"event":"pay","data":"WAIT_BUYER_PAY"}`))
		}
		if pm.Status == "TRADE_SUCCESS" {
			ws.WriteMessage(websocket.TextMessage, []byte(`{"code":200,"event":"pay","data":"TRADE_SUCCESS"}`))
		}
	}
	if pm.Status == "TRADE_BUYER_SIGNED" ||
		pm.Status == "TRADE_SUCCESS" ||
		pm.Status == "TRADE_CLOSED" {
		clientQRMap.Delete(qrID)
		ws.Close()
	}
}
