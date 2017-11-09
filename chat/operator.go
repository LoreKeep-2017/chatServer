package chat

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"time"

	"github.com/LoreKeep-2017/chatServer/auth"

	"golang.org/x/net/websocket"
)

const operatorChannelBufSize = 100

// Chat operator.
type Operator struct {
	Id          int `json:"id"`
	ws          *websocket.Conn
	server      *Server
	rooms       map[int]*Room
	ch          chan ResponseMessage
	doneCh      chan bool
	addToRoomCh chan *Room
}

// Create new chat operator.
func NewOperator(ws *websocket.Conn, server *Server) *Operator {

	if ws == nil {
		panic("ws cannot be nil")
	}

	if server == nil {
		panic("server cannot be nil")
	}

	rooms := make(map[int]*Room)
	ch := make(chan ResponseMessage, channelBufSize)
	doneCh := make(chan bool)
	addToRoomCh := make(chan *Room, channelBufSize)

	return &Operator{0, ws, server, rooms, ch, doneCh, addToRoomCh}
}

func (o *Operator) sendChangeStatus(room Room) {
	jsonstring, err := json.Marshal(room)
	if !CheckError(err, "Invalid RawData", false) {
		msg := ResponseMessage{Action: actionChangeStatusRooms, Status: "Server error", Code: 502}
		websocket.JSON.Send(o.ws, msg)
	}
	msg := ResponseMessage{Action: actionChangeStatusRooms, Status: "OK", Code: 200, Body: jsonstring}
	websocket.JSON.Send(o.ws, msg)
}

func (o *Operator) searchRoomByStatus(typeRoom string) map[int]*Room {
	var rows *sql.Rows
	var err error
	if typeRoom == roomBusy {
		rows, err = o.server.db.Query("SELECT room, description, title, nickname, status, operator FROM room where status=$1 and operator=$2", typeRoom, o.Id)
	} else {
		rows, err = o.server.db.Query("SELECT room, description, title, nickname, status, operator FROM room where status=$1", typeRoom)
	}
	if err != nil {
		panic(err)
	}
	result := make(map[int]*Room, 0)
	for rows.Next() {
		var room int
		var description string
		var title string
		var nickname string
		var status string
		var operator int
		_ = rows.Scan(&room, &description, &title, &nickname, &status, &operator)
		r := Room{Id: room, Status: status, Description: description, Title: title, Operator: &Operator{Id: operator}, Client: &Client{Nick: nickname}}
		result[room] = &r
	}
	return result
}

// Listen Write and Read request via chanel
func (o *Operator) Listen() {
	go o.listenWrite()
	o.listenRead()
}

// Listen write request via chanel
func (o *Operator) listenWrite() {
	log.Println("Listening write to client")
	for {
		select {

		// send message to the operator
		case msg := <-o.ch:
			log.Println(o.ws, msg)
			if o.ws != nil {
				websocket.JSON.Send(o.ws, msg)
			}

		// receive done request
		case <-o.doneCh:
			o.server.DelOperator(o)
			o.doneCh <- true // for listenRead method
			return
		}
	}
}

// Listen read request via chanel
func (o *Operator) listenRead() {
	log.Println("Listening read from client")
	for {
		select {

		// receive done request
		case <-o.doneCh:
			o.server.DelOperator(o)
			o.doneCh <- true // for listenWrite method
			return

		// read data from websocket connection
		default:
			var msg RequestMessage
			err := websocket.JSON.Receive(o.ws, &msg)
			if err == io.EOF {
				o.doneCh <- true
			} else if err != nil {
				o.server.Err(err)
			}
			switch msg.Action {

			//получение всех клиентов
			case actionGetAllRooms:
				log.Println(actionGetAllRooms)
				response := OperatorResponseRooms{o.server.rooms, len(o.server.rooms)}
				jsonstring1, _ := json.Marshal(response)
				log.Println(jsonstring1)
				msg := ResponseMessage{Action: actionGetAllRooms, Status: "OK", Code: 200, Body: jsonstring1}
				o.ch <- msg

			//вход в комнату
			case actionEnterRoom:
				log.Println(actionEnterRoom)
				var rID RequestActionWithRoom
				err := json.Unmarshal(msg.Body, &rID)
				if !CheckError(err, "Invalid RawData"+string(msg.Body), false) {
					msg := ResponseMessage{Action: actionEnterRoom, Status: "Invalid Request", Code: 403}
					o.ch <- msg
				}
				room := o.server.rooms[rID.ID]
				room.Status = roomBusy
				room.Operator = o
				o.rooms[room.Id] = room
				jsonstring, _ := json.Marshal(room)

				_, dberr := o.server.db.Query(`UPDATE operator SET rooms = array_append(rooms,$1) WHERE id=$2`,
					room.Id,
					o.Id,
				)
				_, dberr1 := o.server.db.Query(`UPDATE room SET operator=$2 WHERE room=$1`,
					room.Id,
					o.Id,
				)
				response := ResponseMessage{}
				if dberr != nil || dberr1 != nil {
					response.Action = actionEnterRoom
					response.Status = dberr.Error() + dberr1.Error()
					response.Code = 500
				} else {
					response.Action = actionEnterRoom
					response.Status = "OK"
					response.Code = 200
					response.Body = jsonstring
					room.channelForStatus <- roomBusy
				}
				o.ch <- response
				//msg := ResponseMessage{Action: actionEnterRoom, Status: "OK", Code: 200, Body: jsonstring}
				//o.ch <- msg
				//room.channelForStatus <- roomBusy

			//отправка сообщения
			case actionSendMessage:
				log.Println(actionSendMessage)
				var message Message
				err := json.Unmarshal(msg.Body, &message)
				if !CheckError(err, "Invalid RawData"+string(msg.Body), false) {
					msg := ResponseMessage{Action: actionSendMessage, Status: "Invalid Request", Code: 403}
					o.ch <- msg
				}
				message.Time = int(time.Now().Unix())
				message.Author = "operator"
				room, ok := o.rooms[message.Room]
				if ok {
					room.channelForMessage <- message
				} else {
					msg := ResponseMessage{Action: actionSendMessage, Status: "Room not found", Code: 404}
					o.ch <- msg
				}

			//получение всех сообщений
			case actionGetAllMessages:
				log.Println(actionGetAllMessages)
				var rID RequestActionWithRoom
				err := json.Unmarshal(msg.Body, &rID)
				if !CheckError(err, "Invalid RawData"+string(msg.Body), false) {
					msg := ResponseMessage{Action: actionGetAllMessages, Status: "Invalid Request", Code: 403}
					o.ch <- msg
				}
				messages := make([]Message, 0)
				rows, err := o.server.db.Query("SELECT room, type, date, body FROM message where room=$1", rID.ID)
				if err != nil {
					msg := ResponseMessage{Action: actionGetAllMessages, Status: "Room not found", Code: 404, Body: msg.Body}
					o.ch <- msg
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
					msg := ResponseMessage{Action: actionGetAllMessages, Status: "OK", Code: 200, Body: jsonMessages}
					o.ch <- msg
				}

			//покидание комнаты
			case actionLeaveRoom:
				log.Println(actionLeaveRoom)
				var rID RequestActionWithRoom
				err := json.Unmarshal(msg.Body, &rID)
				if !CheckError(err, "Invalid RawData"+string(msg.Body), false) {
					msg := ResponseMessage{Action: actionLeaveRoom, Status: "Invalid Request", Code: 403}
					o.ch <- msg
				}
				if room, ok := o.rooms[rID.ID]; ok {
					room.Status = roomInProgress
					delete(o.rooms, room.Id)
					_, dberr := o.server.db.Query(`UPDATE operator SET rooms = array_remove(rooms,$1) WHERE id=$2`,
						room.Id,
						o.Id,
					)
					response := ResponseMessage{}
					if dberr != nil {
						response.Action = actionLeaveRoom
						response.Status = dberr.Error()
						response.Code = 500
					} else {
						response.Action = actionLeaveRoom
						response.Status = "OK"
						response.Code = 200
						response.Body = msg.Body
					}
					o.ch <- response
					room.channelForStatus <- roomInProgress
				} else {
					msg := ResponseMessage{Action: actionLeaveRoom, Status: "Room not found", Code: 404, Body: msg.Body}
					o.ch <- msg
				}

			//закрытие комнаты
			case actionCloseRoom:
				log.Println(actionCloseRoom)
				var rID RequestActionWithRoom
				err := json.Unmarshal(msg.Body, &rID)
				if !CheckError(err, "Invalid RawData"+string(msg.Body), false) {
					msg := ResponseMessage{Action: actionCloseRoom, Status: "Invalid Request", Code: 400}
					o.ch <- msg
				}
				if room, ok := o.rooms[rID.ID]; ok {
					room.Status = roomClose
					delete(o.rooms, room.Id)
					_, dberr := o.server.db.Query(`UPDATE operator SET rooms = array_remove(rooms,$1) WHERE id=$2`,
						room.Id,
						o.Id,
					)
					response := ResponseMessage{}
					if dberr != nil {
						response.Action = actionCloseRoom
						response.Status = dberr.Error()
						response.Code = 500
					} else {
						response.Action = actionCloseRoom
						response.Status = "OK"
						response.Code = 200
						response.Body = msg.Body
					}
					o.ch <- response
					room.channelForStatus <- roomClose
				} else {
					msg := ResponseMessage{Action: actionCloseRoom, Status: "Room not found", Code: 404, Body: msg.Body}
					o.ch <- msg
				}

			//получение комнаты по статусу
			case actionGetRoomsByStatus:
				log.Println(actionGetRoomsByStatus)
				var typeRoom RequestTypeRooms
				err := json.Unmarshal(msg.Body, &typeRoom)
				if !CheckError(err, "Invalid RawData"+string(msg.Body), false) {
					msg := ResponseMessage{Action: actionGetRoomsByStatus, Status: "Invalid Request", Code: 400}
					o.ch <- msg
				} else {
					result := o.searchRoomByStatus(typeRoom.Type)
					response := OperatorResponseRooms{result, len(result)}
					rooms, _ := json.Marshal(response)
					msg := ResponseMessage{Action: actionGetRoomsByStatus, Status: "OK", Code: 200, Body: rooms}
					o.ch <- msg
				}

			//получение списка операторов
			case actionGetOperators:
				log.Println(actionGetOperators)
				rows, err := o.server.db.Query("SELECT id, nickname FROM operator")
				if err != nil {
					msg := ResponseMessage{Action: actionGetOperators, Status: "Invalid Request", Code: 400}
					o.ch <- msg
				} else {
					result := make([]Operator, 0)
					for rows.Next() {
						var nickanme string
						var id int
						_ = rows.Scan(&id, &nickanme)
						o := Operator{Id: id}
						result = append(result, o)
					}
					operators, _ := json.Marshal(result)
					msg := ResponseMessage{Action: actionGetOperators, Status: "OK", Code: 200, Body: operators}
					o.ch <- msg
				}

			case actionSendID:
				log.Println(actionSendID)
				var id auth.OperatorId
				err := json.Unmarshal(msg.Body, &id)
				if err == nil {
					o.Id = id.Id
					msg := ResponseMessage{Action: actionSendID, Status: "OK", Code: 200}
					o.ch <- msg
				} else {
					msg := ResponseMessage{Action: actionSendID, Status: "Invalid request", Code: 400}
					o.ch <- msg
				}

			}

		}

	}
}
