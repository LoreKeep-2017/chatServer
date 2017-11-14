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
			if c.ws != nil {
				websocket.JSON.Send(c.ws, msg)
			}

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

				//отправка сообщений
			case actionSendFirstMessage:
				log.Println(actionSendFirstMessage)
				var message Message
				err := json.Unmarshal(msg.Body, &message)
				if !CheckError(err, "Invalid RawData"+string(msg.Body), false) {
					msg := ResponseMessage{Action: actionSendMessage, Status: "Invalid Request", Code: 403}
					c.ch <- msg
				}
				if (c.room != nil) || (c.room.Status != roomBusy) || (c.room.Status != roomClose) {
					message.Author = "client"
					message.Room = c.room.Id
					message.Time = int(time.Now().Unix())
					c.room.Time = int(time.Now().Unix())
					_, err := c.server.db.Query(`update room set description=$1, date=$2 where room=$3`,
						message.Body,
						c.room.Time,
						c.room.Id,
					)
					if err != nil {
						msg := ResponseMessage{Action: actionSendMessage, Status: "db error", Code: 502}
						c.ch <- msg
					}
					c.room.channelForStatus <- roomNew
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
				// log.Println(actionGetAllMessages)
				// messages, _ := json.Marshal(c.room.Messages)
				// response := ResponseMessage{Action: actionGetAllMessages, Status: "OK", Code: 200, Body: messages}
				// log.Println(response)
				messages := make([]Message, 0)
				rows, err := c.server.db.Query("SELECT room, type, date, body FROM message where room=$1", c.room.Id)
				if err != nil {
					msg := ResponseMessage{Action: actionGetAllMessages, Status: "Room not found", Code: 404, Body: msg.Body}
					c.ch <- msg
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
					c.ch <- msg
				}
				//c.ch <- response

			case actionRestoreRoom:
				log.Println(actionRestoreRoom)
				var room ClientRoom
				err := json.Unmarshal(msg.Body, &room)
				if !CheckError(err, "Invalid RawData"+string(msg.Body), false) {
					msg := ResponseMessage{Action: actionRestoreRoom, Status: "Invalid Request", Code: 400}
					c.ch <- msg
				} else {
					r, ok := c.server.rooms[room.RoomID]
					if ok {
						r.Client = c
						c.room = r
						if r.Operator != nil {
							c.server.operators[r.Operator.Id].rooms[r.Id] = r
						}
						c.room.channelForStatus <- roomBusy

						messages := make([]Message, 0)
						rows, err := c.server.db.Query("SELECT room, type, date, body FROM message where room=$1", r.Id)
						if err != nil {
							msg := ResponseMessage{Action: actionGetAllMessages, Status: "Room not found", Code: 404, Body: msg.Body}
							c.ch <- msg
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
							c.ch <- msg
						}
					} else {
						msg := ResponseMessage{Action: actionRestoreRoom, Status: "Room not found", Code: 404}
						c.ch <- msg
					}

				}
			}
		}
	}
}
