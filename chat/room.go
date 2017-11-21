package chat

import (
	"encoding/json"
	"log"
)

const (
	roomChannelBufSize = 100
	//статусы
	roomNotActive  = "roomNotActive"
	roomNew        = "roomNew"
	roomBusy       = "roomBusy"
	roomInProgress = "roomInProgress"
	roomClose      = "roomClose"
	roomSend       = "roomSend"
	roomRecieved   = "roomRecieved"
)

// Chat operator.
type Room struct {
	Id                int `json:"id"`
	channelForMessage chan Message
	channelForStatus  chan string
	server            *Server
	Client            *Client   `json:"client,omitempty"`
	Operator          *Operator `json:"operator,omitempty"`
	Messages          []Message `json:"messages,omitempty"`
	Status            string    `json:"status,omitempty"`
	Description       string    `json:"description,omitempty"`
	LastMessage       string    `json:"lastMessage,omitempty"`
	Time              int       `json:"time"`
	Note              string    `json:"note,omitempty"`
}

// Create new room.
func NewRoom(server *Server) *Room {

	var roomId int
	err := server.db.QueryRow(`insert into room default values returning room;`).Scan(&roomId)
	if err != nil {
		panic(err.Error())
	}

	ch := make(chan Message, roomChannelBufSize)
	channelForStatus := make(chan string, roomChannelBufSize)
	messages := make([]Message, 0)
	status := roomNotActive

	return &Room{
		Id:                roomId,
		channelForMessage: ch,
		channelForStatus:  channelForStatus,
		server:            server,
		Messages:          messages,
		Status:            status}
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
			log.Println("channelForMessage")
			//r.Messages = append(r.Messages, msg)
			_, err := r.server.db.Query(`insert into message(room, type, date, body) values($1, $2, $3, $4)`,
				r.Id,
				msg.Author,
				msg.Time,
				msg.Body,
			)
			_, err = r.server.db.Query(`update room set lastmessage=$1 where room=$2`,
				msg.Body,
				r.Id,
			)
			r.LastMessage = msg.Body
			var response ResponseMessage
			messages := make([]Message, 0)
			rows, err := r.server.db.Query("SELECT room, type, date, body FROM message where room=$1", r.Id)
			if err != nil {
				response = ResponseMessage{Action: actionSendMessage, Status: err.Error(), Code: 404}
			} else {
				for rows.Next() {
					var room int
					var typeM string
					var date int
					var body string
					_ = rows.Scan(&room, &typeM, &date, &body)
					m := Message{typeM, body, room, date}
					messages = append(messages, m)
				}
				jsonMessages, _ := json.Marshal(messages)
				response = ResponseMessage{Action: actionSendMessage, Status: "OK", Code: 200, Body: jsonMessages}
			}
			if msg.Author == "client" && r.Operator != nil {
				r.channelForStatus <- roomRecieved
			}
			log.Println(response)
			if r.Client != nil {
				r.Client.ch <- response
			}
			if r.Operator != nil {
				r.Operator.ch <- response
			}

		//изменения статуса комнаты
		case msg := <-r.channelForStatus:
			r.Status = msg
			jsonstring, _ := json.Marshal(r)
			response := ResponseMessage{}
			_, err := r.server.db.Query(`update room set status=$1 where room=$2`,
				r.Status,
				r.Id)
			if err != nil {
				response.Action = actionChangeStatusRooms
				response.Status = err.Error()
				response.Code = 500
			} else {
				response.Action = actionChangeStatusRooms
				response.Status = "OK"
				response.Code = 200
				response.Body = jsonstring

			}
			log.Println("chsnge status!!!")
			if msg == roomRecieved || msg == roomSend {
				r.server.sendMessageToOperator(r.Operator.Id, actionChangeStatusRooms, jsonstring)
			} else {
				r.server.broadcast(response)
				r.Client.ch <- response
			}

		}
	}
}
