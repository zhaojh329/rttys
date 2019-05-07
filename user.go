package main

type User struct {
	*Client
	sid string
}

type UsrMessage struct {
	msgType int
	data    []byte
	user    *User
}

func (user *User) Close() {
	user.Client.Close()
	user.br.logouting <- user
}

func (user *User) readAlway() {
	defer user.Close()

	for {
		msgType, data, err := user.ws.ReadMessage()
		if err != nil {
			break
		}

		msg := &UsrMessage{msgType, data, user}

		select {
		case user.br.inUsrMessage <- msg:
		case <-user.closeChan:
			return
		}
	}
}
