package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/zhaojh329/rttys/client"

	"net/http"
)

const (
	loginErrorNone    = 0x00
	loginErrorOffline = 0x01
	loginErrorBusy    = 0x02
)

type user struct {
	br         *broker
	sid        string
	devid      string
	conn       *websocket.Conn
	closeMutex sync.Mutex
	closed     bool
	cancel     context.CancelFunc
	send       chan *usrMessage // Buffered channel of outbound messages.
}

type usrMessage struct {
	sid  string
	typ  int
	data []byte
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (u *user) IsDevice() bool {
	return false
}

func (u *user) DeviceID() string {
	return u.devid
}

func (u *user) WriteMsg(typ int, data []byte) {
	u.send <- &usrMessage{
		typ:  typ,
		data: data,
	}
}

func (u *user) Close() {
	defer u.closeMutex.Unlock()

	u.closeMutex.Lock()

	if !u.closed {
		u.closed = true
		u.cancel()
		u.conn.Close()
		u.br.unregister <- u
	}
}

func userLoginAck(code int, c client.Client) {
	msg := fmt.Sprintf(`{"type":"login","err":%d}`, code)
	c.WriteMsg(websocket.TextMessage, []byte(msg))
}

func (u *user) keepAlive(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			u.WriteMsg(websocket.PingMessage, []byte{})
		case <-ctx.Done():
			return
		}
	}
}

func (u *user) readLoop() {
	defer u.Close()

	for {
		typ, data, err := u.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().Msg(err.Error())
			}
			break
		}

		u.br.userMessage <- &usrMessage{u.sid, typ, data}
	}
}

func (u *user) writeLoop(ctx context.Context) {
	defer u.Close()

	for {
		select {
		case msg := <-u.send:
			err := u.conn.WriteMessage(msg.typ, msg.data)
			if err != nil {
				log.Error().Msg(err.Error())
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func serveUser(br *broker, c *gin.Context) {
	devid := c.Param("devid")
	if devid == "" {
		c.Status(http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.Status(http.StatusBadRequest)
		log.Error().Msg(err.Error())
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	u := &user{
		br:     br,
		conn:   conn,
		devid:  devid,
		cancel: cancel,
		send:   make(chan *usrMessage, 256),
	}

	go u.readLoop()
	go u.keepAlive(ctx)
	go u.writeLoop(ctx)

	br.register <- u
}
