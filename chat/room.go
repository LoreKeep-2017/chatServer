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

//Send message
// func (r Room) sendMessage(response ResponseMessage) {
// 	websocket.JSON.Send(r.Operator.ws, response)
// 	websocket.JSON.Send(r.Client.ws, response)
// }

// Listen write request via chanel
func (r *Room) listenWrite() {
	log.Println("Listening write to room")
	for {
		select {

		// отправка сообщений участникам комнаты
		case msg := <-r.channelForMessage:
			r.Messages = append(r.Messages, msg)
			messages, _ := json.Marshal(r.Messages)
			response := ResponseMessage{Action: actionSendMessage, Status: "OK", Room: r.Id, Code: 200, Body: messages}
			log.Println(response)
			r.Client.ch <- response
			r.Operator.ch <- response
			//websocket.JSON.Send(r.Operator.ws, response)
			//websocket.JSON.Send(r.Client.ws, response)
			//r.sendMessage(response)

			//добавление описание комнате
		case description := <-r.channelForDescription:
			r.Description = description.Description
			r.Title = description.Title
			r.Status = roomNew
			msg, _ := json.Marshal(r)
			response := ResponseMessage{Action: actionChangeStatusRooms, Status: "OK", Code: 200, Body: msg}
			r.Client.ch <- response
			r.server.broadcastChangeStatus(*r)

		//изменения статуса комнаты
		case msg := <-r.channelForStatus:
			log.Println("change status", msg, r.Description)
			r.Status = msg
			jsonstring1, _ := json.Marshal(r)
			response := ResponseMessage{Action: actionChangeStatusRooms, Status: "OK", Code: 200, Body: jsonstring1}
			websocket.JSON.Send(r.Client.ws, response)
			r.server.broadcastChangeStatus(*r)
		}
	}
}
