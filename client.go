package main

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	br    *Broker
	ws    *websocket.Conn
	devid string

	mutex     sync.Mutex /* Avoid repeated closes */
	closed    bool
	closeChan chan byte

	outMessage chan *wsOutMessage /* Buffered channel of outbound messages */
}

type wsOutMessage struct {
	msgType int
	data    []byte
}

func (c *Client) Close() {
	defer c.mutex.Unlock()
	c.mutex.Lock()

	if !c.closed {
		c.ws.Close()
		c.closed = true
		close(c.closeChan)
	}
}

func (c *Client) wsWrite(msgType int, data []byte) {
	c.outMessage <- &wsOutMessage{msgType, data}
}

func (c *Client) writePump() {
	defer c.Close()

	for {
		select {
		case msg := <-c.outMessage:
			if err := c.ws.WriteMessage(msg.msgType, msg.data); err != nil {
				return
			}
		case <-c.closeChan:
			return
		}
	}
}

/* serveWs handles websocket requests from the device or user. */
func serveWs(br *Broker, w http.ResponseWriter, r *http.Request, cfg *RttysConfig) {
	isDev := r.URL.Query().Get("device") != ""

	if isDev {
		token := r.Header.Get("Authorization")
		if token != cfg.token {
			log.Error("Invalid token from terminal device")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
	} else if _, ok := httpSessions.Get(r.URL.Query().Get("sid")); !ok {
		log.Error("Invalid sid from client")
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	keepalive, _ := strconv.Atoi(r.URL.Query().Get("keepalive"))
	devid := r.URL.Query().Get("devid")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	if devid == "" {
		conn.Close()
		log.Error("devid required")
		return
	}

	client := &Client{
		br:         br,
		devid:      devid,
		ws:         conn,
		closeChan:  make(chan byte),
		outMessage: make(chan *wsOutMessage, 1000),
	}

	if isDev {
		desc := r.URL.Query().Get("description")
		sessions := make(map[uint8]string)

		dev := &Device{client, desc, time.Now().Unix(), sessions}

		if keepalive > 0 {
			go dev.keepAlive(int64(keepalive))
		}

		go dev.readAlway()

		br.connecting <- dev
	} else {
		user := &User{client, ""}

		go user.readAlway()

		br.logining <- user
	}

	go client.writePump()
}
