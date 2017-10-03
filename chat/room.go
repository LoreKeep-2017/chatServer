package chat

import (
	"encoding/json"
	"log"

	"golang.org/x/net/websocket"
)

const roomChannelBufSize = 1

var roomId int = 0

// Chat operator.
type Room struct {
	id                int
	channelForMessage chan Message
	client            *Client
	operator          *Operator
	messages          []Message
}

// Create new room.
func NewRoom(client *Client, operator *Operator) *Room {

	if client == nil {
		panic("client cannot be nil")
	}

	if operator == nil {
		panic("operator cannot be nil")
	}

	roomId++
	ch := make(chan Message, roomChannelBufSize)
	messages := make([]Message, 0)

	return &Room{roomId, ch, client, operator, messages}
}

// Listen Write and Read request via chanel
func (r *Room) Listen() {
	go r.listenWrite()
}

// Listen write request via chanel
func (r *Room) listenWrite() {
	log.Println("Listening write to room")
	for {
		select {

		// отправка сообщений участникам комнаты
		case msg := <-r.channelForMessage:
			r.messages = append(r.messages, msg)
			messages, _ := json.Marshal(r.messages)
			msg1 := ResponseMessage{Action: actionSendMessage, Status: "OK", Code: 200, Body: messages}
			websocket.JSON.Send(r.operator.ws, msg1)
			websocket.JSON.Send(r.client.ws, msg1)

		}
	}
}
