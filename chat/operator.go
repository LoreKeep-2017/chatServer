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
	id           int
	ws           *websocket.Conn
	server       *Server
	rooms        map[int]*Room
	ch           chan *Message
	freeClientCh chan bool
	doneCh       chan bool
	addToRoomCh  chan *Room
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
	ch := make(chan *Message, channelBufSize)
	freeClientCh := make(chan bool, channelBufSize)
	doneCh := make(chan bool)
	addToRoomCh := make(chan *Room, channelBufSize)

	return &Operator{operatorId, ws, server, rooms, ch, freeClientCh, doneCh, addToRoomCh}
}

func (o *Operator) sendAllClients() {
	o.freeClientCh <- true

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

		// send message to the client
		case msg := <-o.ch:
			//log.Println("Send:", msg)
			//websocket.JSON.Send(o.ws, msg)
			messages, _ := json.Marshal(msg)
			msg1 := ResponseMessage{Action: actionSendMessage, Status: "OK", Code: 200, Body: messages}
			websocket.JSON.Send(o.ws, msg1)

		// send  all free clients
		case send := <-o.freeClientCh:
			if send {
				jsonstring, _ := json.Marshal(o.server.clients)
				msg1 := ResponseMessage{Action: actionGetAllClients, Status: "OK", Code: 200, Body: jsonstring}
				websocket.JSON.Send(o.ws, msg1)
			}

		// adding to room
		case room := <-o.addToRoomCh:
			o.rooms[room.id] = room
			response := OperatorResponseAddToRoom{room.id}
			jsonstring, _ := json.Marshal(response)
			msg := ResponseMessage{Action: actionCreateRoom, Status: "OK", Code: 200, Body: jsonstring}
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
			case actionGetAllClients:
				log.Println(actionGetAllClients)
				jsonstring, _ := json.Marshal(o.server.clients)
				msg := ResponseMessage{Action: actionGetAllClients, Status: "OK", Code: 200, Body: jsonstring}
				websocket.JSON.Send(o.ws, msg)

			//создание комнаты
			case actionCreateRoom:
				log.Println(actionCreateRoom)
				var cid RequestCreateRoom
				err := json.Unmarshal(msg.Body, &cid)
				if !CheckError(err, "Invalid RawData"+string(msg.Body), false) {
					msg := ResponseMessage{Action: actionCreateRoom, Status: "Invalid Request", Code: 403}
					websocket.JSON.Send(o.ws, msg)
					//return
				}
				msg := ResponseMessage{Action: actionCreateRoom, Status: "OK", Code: 200}
				websocket.JSON.Send(o.ws, msg)
				o.server.CreateRoom(cid.ID, o)

			//отправка сообщения
			case actionSendMessage:
				log.Println(actionSendMessage)
				var message Message
				err := json.Unmarshal(msg.Body, &message)
				if !CheckError(err, "Invalid RawData"+string(msg.Body), false) {
					msg := ResponseMessage{Action: actionSendMessage, Status: "Invalid Request", Code: 403}
					websocket.JSON.Send(o.ws, msg)
					//return
				}
				message.Time = int(time.Now().Unix())
				message.Author = msg.Type
				room, ok := o.rooms[message.Room]
				if ok {
					// messages, _ := json.Marshal(room.messages)
					// msg := ResponseMessage{Action: actionSendMessage, Status: "OK", Code: 200, Body: messages}
					// websocket.JSON.Send(o.ws, msg)
					room.channelForMessage <- message
				} else {
					msg := ResponseMessage{Action: actionSendMessage, Status: "Room not found", Code: 404}
					websocket.JSON.Send(o.ws, msg)
				}

			//закрытие комнаты
			case actionCloseRoom:
				log.Println(actionCloseRoom)
				//var message Message

			}

		}

	}
}
