package chat

import (
	"encoding/json"
	"io"
	"log"
	"time"

	"golang.org/x/net/websocket"
)

const operatorChannelBufSize = 100

var operatorId int = 0

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

	operatorId++
	rooms := make(map[int]*Room)
	ch := make(chan ResponseMessage, channelBufSize)
	doneCh := make(chan bool)
	addToRoomCh := make(chan *Room, channelBufSize)

	return &Operator{operatorId, ws, server, rooms, ch, doneCh, addToRoomCh}
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
			websocket.JSON.Send(o.ws, msg)

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

				msg := ResponseMessage{Action: actionEnterRoom, Status: "OK", Code: 200, Body: jsonstring}
				o.ch <- msg
				room.channelForStatus <- roomBusy

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
					msg := ResponseMessage{Action: actionLeaveRoom, Status: "OK", Code: 200, Body: msg.Body}
					o.ch <- msg
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
					msg := ResponseMessage{Action: actionCloseRoom, Status: "OK", Code: 200, Body: msg.Body}
					o.ch <- msg
					room.channelForStatus <- roomClose
				} else {
					msg := ResponseMessage{Action: actionCloseRoom, Status: "Room not found", Code: 404, Body: msg.Body}
					o.ch <- msg
				}

			}

		}

	}
}
