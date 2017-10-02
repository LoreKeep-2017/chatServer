package chat

import (
	"encoding/json"
	"fmt"
	"io"
	"log"

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
	freeClientCh chan *map[int]*Client
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
	freeClientCh := make(chan *map[int]*Client, channelBufSize)
	doneCh := make(chan bool)
	addToRoomCh := make(chan *Room, channelBufSize)

	return &Operator{operatorId, ws, server, rooms, ch, freeClientCh, doneCh, addToRoomCh}
}

func (o *Operator) sendAllClients(c map[int]*Client) {
	select {
	case o.freeClientCh <- &c:
	default:
		o.server.DelOperator(o)
		err := fmt.Errorf("operator %d is disconnected.", o.id)
		o.server.Err(err)
	}
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
			log.Println("Send:", msg)
			websocket.JSON.Send(o.ws, msg)

		// send  all free clients
		case msg := <-o.freeClientCh:
			log.Println("Send:", msg)
			websocket.JSON.Send(o.ws, msg)

		// adding to room
		case room := <-o.addToRoomCh:
			o.rooms[room.id] = room
			response := OperatorResponseAddToRoom{"add to room", room.id}
			websocket.JSON.Send(o.ws, response)

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
			var msg OperatorRequest
			err := websocket.JSON.Receive(o.ws, &msg)
			if err == io.EOF {
				o.doneCh <- true
			} else if err != nil {
				o.server.Err(err)
			}
			log.Println(msg)
			switch msg.Action {
			case "1":
				log.Println("1")
				websocket.JSON.Send(o.ws, o.server.clients)
			case "2":
				log.Println("2")
				var cid OperatorGrabb
				err := json.Unmarshal(msg.RawData, &cid)
				if !CheckError(err, "Invalid RawData"+string(msg.RawData), false) {
					return
				}
				log.Println(cid)
				o.server.CreateRoom(cid.Id, o)
			case "3":
				log.Println("3")
				var message Message
				err := json.Unmarshal(msg.RawData, &message)
				if !CheckError(err, "Invalid RawData"+string(msg.RawData), false) {
					return
				}
				log.Println(message)
				room, ok := o.rooms[message.Room]
				if ok {
					room.channelForMessage <- message
				} else {
					msg := ClientGreetingResponse{"404", "room not found:)"}
					websocket.JSON.Send(o.ws, msg)
				}

			}

		}

	}
}
