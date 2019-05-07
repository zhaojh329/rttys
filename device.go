package main

import (
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"time"
)

const (
	/* Max session id for each device */
	RTTY_MAX_SESSION_ID_DEV = 5
)

type Device struct {
	*Client
	desc      string           /* description of the device */
	timestamp int64            /* Connection time */
	sessions  map[uint8]string /* sessions of each device */
}

type DeviceInfo struct {
	ID          string `json:"id"`
	Uptime      int64  `json:"uptime"`
	Description string `json:"description"`
}

type DevMessage struct {
	msgType int
	data    []byte
	dev     *Device
}

func (dev *Device) Close() {
	dev.Client.Close()
	dev.br.disconnecting <- dev
}

func (dev *Device) getFreeSid() uint8 {
	for sid := uint8(1); sid <= RTTY_MAX_SESSION_ID_DEV; sid++ {
		if _, ok := dev.sessions[sid]; !ok {
			return sid
		}
	}
	return uint8(0)
}

/*
 * If the Server does not receive a PING Packet from the Client within one and
 * a half times the Keep Alive time period, the server will disconnect the
 * Connection
 */
func (dev *Device) keepAlive(keepalive int64) {
	defer dev.Close()

	ticker := time.NewTicker(time.Second * time.Duration(keepalive))
	defer ticker.Stop()

	last := time.Now().Unix()
	keepalive = keepalive * 3

	/* Get the current ping handler */
	pingHandler := dev.ws.PingHandler()

	dev.ws.SetPingHandler(func(appData string) error {
		last = time.Now().Unix()
		return pingHandler(appData)
	})

	for {
		select {
		case <-dev.closeChan:
			return
		case <-ticker.C:
			now := time.Now().Unix()
			if now-last > keepalive {
				log.Error("Inactive device in long time, now kill it: %s", dev.devid)
				return
			}
		}
	}
}

func (dev *Device) readAlway() {
	defer dev.Close()

	for {
		msgType, data, err := dev.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("error: %v", err)
			}
			break
		}

		msg := &DevMessage{msgType, data, dev}

		select {
		case dev.br.inDevMessage <- msg:
		case <-dev.closeChan:
			log.Error("closeChan from readAlway")
			return
		}
	}
}
