package chat

import (
	"encoding/json"
	"io"
	"log"
	"time"

	"golang.org/x/net/websocket"
)

const channelBufSize = 100

var maxId int = 0

// Chat client.
type Client struct {
	Id        int    `json:"id"`
	Nick      string `json:"nick"`
	ws        *websocket.Conn
	server    *Server
	room      *Room
	ch        chan ResponseMessage
	doneCh    chan bool
	addRoomCh chan *Room
	delRoomCh chan *Room
}

// Create new chat client.
func NewClient(ws *websocket.Conn, server *Server, nick string, room *Room) *Client {

	if ws == nil {
		panic("ws cannot be nil")
	}

	if server == nil {
		panic("server cannot be nil")
	}

	maxId++
	ch := make(chan ResponseMessage, channelBufSize)
	doneCh := make(chan bool)
	addRoomCh := make(chan *Room)
	delRoomCh := make(chan *Room)
	return &Client{maxId, nick, ws, server, room, ch, doneCh, addRoomCh, delRoomCh}
}

func (c *Client) Conn() *websocket.Conn {
	return c.ws
}

func (c *Client) Done() {
	c.doneCh <- true
}

// Listen Write and Read request via chanel
func (c *Client) Listen() {
	go c.listenWrite()
	c.listenRead()
}

// Listen write request via chanel
func (c *Client) listenWrite() {
	log.Println("Listening write to client")
	for {
		select {

		// send message to the client
		case msg := <-c.ch:
			log.Println("Send:", msg, c.ws)
			websocket.JSON.Send(c.ws, msg)

		// receive done request
		case <-c.doneCh:
			c.server.Del(c)
			c.doneCh <- true // for listenRead method
			return

		}

	}
}

// Listen read request via chanel
func (c *Client) listenRead() {
	log.Println("Listening read from client")
	for {
		select {

		// receive done request
		case <-c.doneCh:
			c.server.Del(c)
			c.doneCh <- true // for listenWrite method
			return

		// read data from websocket connection
		default:
			var msg RequestMessage
			err := websocket.JSON.Receive(c.ws, &msg)
			if err == io.EOF {
				c.doneCh <- true
			} else if err != nil {
				c.server.Err(err)
			}
			log.Println(msg)
			switch msg.Action {

			//отправка сообщений
			case actionSendMessage:
				log.Println(actionSendMessage)
				var message Message
				err := json.Unmarshal(msg.Body, &message)
				if !CheckError(err, "Invalid RawData"+string(msg.Body), false) {
					msg := ResponseMessage{Action: actionSendMessage, Status: "Invalid Request", Code: 403}
					c.ch <- msg
				}
				if c.room != nil {
					message.Author = "client"
					message.Room = c.room.Id
					message.Time = int(time.Now().Unix())
					c.room.channelForMessage <- message
				} else {
					msg := ResponseMessage{Action: actionSendMessage, Status: "Room not found", Code: 404}
					c.ch <- msg
				}

			//описание комнаты
			case actionSendDescriptionRoom:
				log.Println(actionSendDescriptionRoom)
				var roomDescription ClientSendDescriptionRoomRequest
				err := json.Unmarshal(msg.Body, &roomDescription)
				if !CheckError(err, "Invalid RawData"+string(msg.Body), false) {
					msg := ResponseMessage{Action: actionSendDescriptionRoom, Status: "Invalid Request", Code: 403}
					c.ch <- msg
				} else {
					c.Nick = roomDescription.Nick
					c.room.channelForDescription <- roomDescription
				}

			//закрытие комнаты
			case actionCloseRoom:
				log.Println(actionCloseRoom)
				c.room.Status = roomClose
				c.room.channelForStatus <- roomClose

			//получение всех сообщений
			case actionGetAllMessages:
				log.Println(actionGetAllMessages)
				messages, _ := json.Marshal(c.room.Messages)
				response := ResponseMessage{Action: actionGetAllMessages, Status: "OK", Code: 200, Body: messages}
				log.Println(response)
				c.ch <- response
			}
		}
	}
}
