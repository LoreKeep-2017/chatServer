package chat

import (
	"database/sql"
	"encoding/json"
	"log"
	"strconv"
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
	//file dir
	fileDir = "/home/tp/fileChat/"
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

	second, err := server.db.Query(`update room set nickname=$1 where room=$2`,
		"User_"+strconv.Itoa(roomId),
		roomId,
	)
	if err != nil {
		second.Close()
	}
	second.Close()

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
			first, err := r.server.db.Query(`insert into message(room, type, date, body, url) values($1, $2, $3, $4, $5)`,
				r.Id,
				msg.Author,
				msg.Time,
				msg.Body,
				msg.ImageUrl,
			)
			first.Close()
			second, err := r.server.db.Query(`update room set lastmessage=$1 where room=$2`,
				msg.Body,
				r.Id,
			)
			second.Close()
			r.LastMessage = msg.Body
			var response ResponseMessage
			messages := make([]Message, 0)
			rows, err := r.server.db.Query("SELECT room, type, date, body, url FROM message where room=$1", r.Id)
			if err != nil {
				rows.Close()
				response = ResponseMessage{Action: actionSendMessage, Status: err.Error(), Code: 404}
			} else {
				for rows.Next() {
					var room sql.NullInt64
					var typeM sql.NullString
					var date sql.NullInt64
					var body sql.NullString
					var url sql.NullString
					_ = rows.Scan(&room, &typeM, &date, &body, &url)
					m := Message{Author: typeM.String, Body: body.String, Room: int(room.Int64), Time: int(date.Int64), ImageUrl: url.String}
					messages = append(messages, m)
				}
				rows.Close()
				// try new struct
				tmp := make([]Message, len(messages))
				copy(tmp, messages)
				r.Messages = tmp
				// try new struct
				jsonMessages, _ := json.Marshal(r)
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
			rows, err := r.server.db.Query(`update room set status=$1 where room=$2`,
				r.Status,
				r.Id)
			rows.Close()
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
