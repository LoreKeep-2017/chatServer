package chat

import (
	"encoding/json"
	"log"

	"golang.org/x/net/websocket"
)

const (
	roomChannelBufSize = 1
	//статусы
	roomNotActive  = "roomNotActive"
	roomNew        = "roomNew"
	roomBusy       = "roomBusy"
	roomInProgress = "roomInProgress"
	roomClose      = "roomClose"
)

var roomId int = 0

// Chat operator.
type Room struct {
	Id                    int `json:"id"`
	channelForMessage     chan Message
	channelForDescription chan ClientSendDescriptionRoomRequest
	channelForStatus      chan string
	server                *Server
	Client                *Client   `json:"client,omitempty"`
	Operator              *Operator `json:"operator,omitempty"`
	Messages              []Message `json:"messages"`
	Status                string    `json:"status,omitempty"`
	Description           string    `json:"description,omitempty"`
	Title                 string    `json:"title,omitempty"`
}

// Create new room.
func NewRoom(server *Server) *Room {

	roomId++
	ch := make(chan Message, roomChannelBufSize)
	channelForDescription := make(chan ClientSendDescriptionRoomRequest)
	channelForStatus := make(chan string)
	messages := make([]Message, 0)
	status := roomNotActive

	return &Room{
		Id:                    roomId,
		channelForMessage:     ch,
		channelForStatus:      channelForStatus,
		server:                server,
		channelForDescription: channelForDescription,
		Messages:              messages,
		Status:                status}
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
			r.Messages = append(r.Messages, msg)
			messages, _ := json.Marshal(r.Messages)
			msg1 := ResponseMessage{Action: actionSendMessage, Status: "OK", Code: 200, Body: messages}
			websocket.JSON.Send(r.Operator.ws, msg1)
			websocket.JSON.Send(r.Client.ws, msg1)

		//добавление описание комнате
		case msg := <-r.channelForDescription:
			log.Println("create description", msg, r.Description)
			r.Description = msg.Description
			r.Title = msg.Title
			r.Status = roomNew
			log.Println(r.Description)
			r.server.broadcastRooms()

		//изменения статуса комнаты
		case msg := <-r.channelForStatus:
			log.Println("change status", msg, r.Description)
			r.Status = msg
			jsonstring, _ := json.Marshal(r)
			response := ResponseMessage{Action: actionChangeStatusRooms, Status: "OK", Code: 200, Body: jsonstring}
			websocket.JSON.Send(r.Client.ws, response)
			r.server.broadcastChangeStatus(*r)
		}
	}
}
