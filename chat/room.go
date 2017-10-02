package chat

import (
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
	messages := make([]Message, 10)

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

		// send message to the client
		case msg := <-r.channelForMessage:
			r.messages = append(r.messages, msg)
			log.Println(msg)
			//отправка сообщений
			//TODO: добавить этот функционал в модели клиента и сервера
			if msg.Author == "client" {
				log.Println("to operator")
				websocket.JSON.Send(r.operator.ws, msg)
			} else {
				log.Println("to client")
				websocket.JSON.Send(r.client.ws, msg)
			}

		}
	}
}
