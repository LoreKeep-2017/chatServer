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
	Nickname    string `json:"nickname"`
	Fio         string `json:"fio"`
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

	return &Operator{0, ws, server, rooms, ch, doneCh, addToRoomCh, " ", " "}
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

func (o *Operator) searchRoomByStatus(typeRoom string) map[int]Room {
	var rows *sql.Rows
	var err error
	if typeRoom == roomBusy || typeRoom == roomSend || typeRoom == roomRecieved {
		rows, err = o.server.db.Query("SELECT room, description, date, status, nickname, operator FROM room where status=$1 and operator=$2", typeRoom, o.Id)
	} else {
		rows, err = o.server.db.Query("SELECT room, description, date, status, nickname, operator FROM room where status=$1", typeRoom)
	}
	if err != nil {
		panic(err)
	}
	result := make(map[int]Room, 0)
	for rows.Next() {
		var room int
		var description string
		var nickname string
		var status string
		var operator int
		var date int
		log.Println()
		_ = rows.Scan(&room, &description, &date, &nickname, &status, &operator)
		log.Println(date)
		r := Room{Id: room, Status: status, Time: date, Description: description, Operator: &Operator{Id: operator}, Client: &Client{Nick: nickname}}
		log.Println(r.Time, r.Id, r.Status)
		result[room] = r
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
				log.Println("if clause")
				if room, ok := o.server.rooms[rID.ID]; ok {
					log.Println(ok)
					room.Status = roomRecieved
					room.Operator = o
					o.rooms[room.Id] = room
					jsonstring, _ := json.Marshal(room)
					log.Println(room)
					log.Println(o.server.db)

					_, dberr := o.server.db.Query(`UPDATE operator SET rooms = array_append(rooms,$1) WHERE id=$2`,
						room.Id,
						o.Id,
					)
					_, dberr1 := o.server.db.Query(`UPDATE room SET operator=$2 WHERE room=$1`,
						room.Id,
						o.Id,
					)
					response := ResponseMessage{}
					if dberr != nil {
						response.Action = actionEnterRoom
						response.Status = dberr.Error()
						response.Code = 500
					} else if dberr1 != nil {
						response.Action = actionEnterRoom
						response.Status = dberr1.Error()
						response.Code = 500
					} else {
						response.Action = actionEnterRoom
						response.Status = "OK"
						response.Code = 200
						response.Body = jsonstring
					}
					if response.Code == 200 {
						room.channelForStatus <- roomRecieved
					}
					o.ch <- response
				} else {
					msg := ResponseMessage{Action: actionEnterRoom, Status: "Room not found", Code: 404}
					o.ch <- msg
				}

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
					log.Println(message)
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

				//
			case actionRoomStatusSend:
				log.Println(actionRoomStatusSend)
				var rID RequestActionWithRoom
				err := json.Unmarshal(msg.Body, &rID)
				if !CheckError(err, "Invalid RawData"+string(msg.Body), false) {
					msg := ResponseMessage{Action: actionRoomStatusSend, Status: "Invalid Request", Code: 400}
					o.ch <- msg
				}
				if room, ok := o.rooms[rID.ID]; ok {
					room.Status = roomSend
					response := ResponseMessage{}

					response.Action = actionRoomStatusSend
					response.Status = "OK"
					response.Code = 200
					response.Body = msg.Body

					o.ch <- response
					room.channelForStatus <- roomSend
				} else {
					msg := ResponseMessage{Action: actionRoomStatusSend, Status: "Room not found", Code: 404, Body: msg.Body}
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
					response := OperatorResponseRoomsNew{result, len(result)}
					rooms, _ := json.Marshal(response)
					msg := ResponseMessage{Action: actionGetRoomsByStatus, Status: "OK", Code: 200, Body: rooms}
					o.ch <- msg
				}

			//получение списка операторов
			case actionGetOperators:
				log.Println(actionGetOperators)
				// _, err := o.server.db.Query("SELECT id, nickname, fio FROM operator")
				// if err != nil {
				// 	msg := ResponseMessage{Action: actionGetOperators, Status: "Invalid Request", Code: 400}
				// 	o.ch <- msg
				// } else {
				result := make([]Operator, 0)
				for _, v := range o.server.operators {
					result = append(result, *v)
				}
				// for rows.Next() {
				// 	var nickanme string
				// 	var fio string
				// 	var id int
				// 	_ = rows.Scan(&id, &nickanme, &fio)
				// 	o := Operator{Id: id, Nickname: nickanme, Fio: fio}
				// 	result = append(result, o)
				// }
				operators, _ := json.Marshal(result)
				msg := ResponseMessage{Action: actionGetOperators, Status: "OK", Code: 200, Body: operators}
				o.ch <- msg
				// }

			case actionSendID:
				log.Println(actionSendID)
				var id auth.OperatorId
				err := json.Unmarshal(msg.Body, &id)
				if err == nil {
					o.Id = id.Id
					o.Fio = id.FIO
					o.Nickname = id.Login
					o.server.AddOperator(o)
					msg := ResponseMessage{Action: actionSendID, Status: "OK", Code: 200}
					o.ch <- msg
				} else {
					msg := ResponseMessage{Action: actionSendID, Status: "Invalid request", Code: 400}
					o.ch <- msg
				}

			case actionChangeOperator:
				log.Println(actionChangeOperator)
				var operatorChange OperatorChange
				err := json.Unmarshal(msg.Body, &operatorChange)
				if err == nil {
					//

					_, dberr := o.server.db.Query(`UPDATE operator SET rooms = array_remove(rooms,$1) WHERE id=$2`,
						operatorChange.Room,
						o.Id,
					)
					_, dberr1 := o.server.db.Query(`UPDATE operator SET rooms = array_append(rooms,$1) WHERE id=$2`,
						operatorChange.Room,
						operatorChange.ID,
					)
					_, dberr2 := o.server.db.Query(`UPDATE room SET operator=$2 WHERE room=$1`,
						operatorChange.Room,
						operatorChange.ID,
					)
					response := ResponseMessage{}
					if (dberr != nil) || (dberr1 != nil) || (dberr2 != nil) {
						response.Action = actionChangeOperator
						response.Status = "db error!"
						response.Code = 500
					} else {
						response.Action = actionChangeOperator
						response.Status = "OK"
						response.Code = 200
						response.Body = msg.Body
					}
					o.ch <- response
					room := o.server.rooms[operatorChange.Room]
					o.server.operators[operatorChange.ID].rooms[room.Id] = room
					delete(o.rooms, room.Id)
					room.Operator = o.server.operators[operatorChange.ID]
					o.rooms[room.Id] = room
					jsonstring, _ := json.Marshal(room)
					o.server.sendMessageToOperator(operatorChange.ID, actionEnterRoom, jsonstring)
					//
				} else {
					msg := ResponseMessage{Action: actionSendID, Status: "Invalid request", Code: 400}
					o.ch <- msg
				}

			}

		}

	}
}
