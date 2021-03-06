package chat

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"

	"github.com/LoreKeep-2017/chatServer/db"
	//"golang.org/x/net/websocket"
)

const channelBufSize = 100

var maxId int = 0

// Chat client.
type Client struct {
	Id        int    `json:"id,omitempty"`
	Nick      string `json:"nick,omitempty"`
	ws        *websocket.Conn
	server    *Server
	room      *Room
	ch        chan ResponseMessage
	doneCh    chan bool
	addRoomCh chan *Room
	delRoomCh chan *Room
}

// Create new chat client.
func NewClient(ws *websocket.Conn, server *Server, room *Room) *Client {

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
	return &Client{Id: maxId, ws: ws, server: server, room: room, ch: ch, doneCh: doneCh, addRoomCh: addRoomCh, delRoomCh: delRoomCh}
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
				websocket.WriteJSON(c.ws, msg)
				//websocket.JSON.Send(c.ws, msg)
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
			err := websocket.ReadJSON(c.ws, &msg)
			//err := websocket.JSON.Receive(c.ws, &msg)
			if err == io.EOF {
				log.Println(err.Error())
				c.doneCh <- true
			} else if err != nil {
				log.Println(err.Error())
				c.doneCh <- true
				//c.server.Err(err)
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
					if message.Image != "" {
						if message.ImageFormat == "" {
							msg := ResponseMessage{Action: actionSendMessage, Status: "Bad request, image format must be jpg/jpeg/svg/png/gif", Code: 400}
							c.ch <- msg
							break
						}
						if _, ok := FormatsImage[message.ImageFormat]; !ok {
							msg := ResponseMessage{Action: actionSendMessage, Status: "Bad request, image format must be jpg/jpeg/svg/png/gif", Code: 400}
							c.ch <- msg
							break
						}
						fileDBurl := fmt.Sprintf("%d.%s", time.Now().UnixNano(), message.ImageFormat)
						fileUrl := fileDir + strconv.Itoa(c.room.Id) + "/" + fileDBurl
						// osUser, err := user.Lookup("tp")
						// osUid, err := strconv.Atoi(osUser.Uid)
						// grUid, err := strconv.Atoi(osUser.Gid)
						if _, err := os.Stat(fileDir + strconv.Itoa(c.room.Id)); os.IsNotExist(err) {
							os.Mkdir(fileDir+strconv.Itoa(c.room.Id), 0777)
							//os.Chown(fileDir+strconv.Itoa(c.room.Id), osUid, grUid)
							//os.Chmod(fileDir+strconv.Itoa(c.room.Id), 7777)
						}
						f, err := os.OpenFile(fileUrl, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
						//os.Chown(fileUrl, osUid, grUid)
						//os.Chmod(fileUrl, 7777)
						if err != nil {
							msg := ResponseMessage{Action: actionSendMessage, Status: "Save image error", Code: 500}
							c.ch <- msg
							break
						} else {
							err := convertString(message.Image, message.ImageFormat, f)
							if err != nil {
								msg := ResponseMessage{Action: actionSendMessage, Status: "Save image error", Code: 500}
								c.ch <- msg
								break
							}
							err = f.Close()
							if err != nil {
								msg := ResponseMessage{Action: actionSendMessage, Status: "Save image error", Code: 500}
								c.ch <- msg
								break
							}
							//_, err = f.Write([]byte(img))
							message.ImageUrl = fileDBurl
						}
					}
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
					msg := ResponseMessage{Action: actionSendFirstMessage, Status: "Invalid Request", Code: 403}
					c.ch <- msg
				}
				if (c.room != nil) || (c.room.Status != roomRecieved) || (c.room.Status != roomSend) {
					message.Author = "client"
					message.Room = c.room.Id
					message.Time = int(time.Now().Unix())
					c.room.Time = int(time.Now().Unix())
					c.room.LastMessage = message.Body
					if message.Image != "" {
						if message.ImageFormat == "" {
							msg := ResponseMessage{Action: actionSendMessage, Status: "Bad request, image format must be jpg/jpeg/svg/png/gif", Code: 400}
							c.ch <- msg
							break
						}
						if _, ok := FormatsImage[message.ImageFormat]; !ok {
							msg := ResponseMessage{Action: actionSendMessage, Status: "Bad request, image format must be jpg/jpeg/svg/png/gif", Code: 400}
							c.ch <- msg
							break
						}
						fileDBurl := fmt.Sprintf("%d.%s", time.Now().UnixNano(), message.ImageFormat)
						fileUrl := fileDir + strconv.Itoa(c.room.Id) + "/" + fileDBurl
						// osUser, err := user.Lookup("tp")
						// osUid, err := strconv.Atoi(osUser.Uid)
						// grUid, err := strconv.Atoi(osUser.Gid)
						if _, err := os.Stat(fileDir + strconv.Itoa(c.room.Id)); os.IsNotExist(err) {
							os.Mkdir(fileDir+strconv.Itoa(c.room.Id), 0777)
							//os.Chown(fileDir+strconv.Itoa(c.room.Id), osUid, grUid)
							//os.Chmod(fileDir+strconv.Itoa(c.room.Id), 7777)
						}
						f, err := os.OpenFile(fileUrl, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
						//os.Chown(fileUrl, osUid, grUid)
						//os.Chmod(fileUrl, 7777)
						if err != nil {
							msg := ResponseMessage{Action: actionSendMessage, Status: "Save image error: " + err.Error(), Code: 500}
							c.ch <- msg
							break
						} else {
							err := convertString(message.Image, message.ImageFormat, f)
							if err != nil {
								msg := ResponseMessage{Action: actionSendMessage, Status: "Save image error: " + err.Error(), Code: 500}
								c.ch <- msg
								break
							}
							err = f.Close()
							if err != nil {
								msg := ResponseMessage{Action: actionSendMessage, Status: "Save image error: " + err.Error(), Code: 500}
								c.ch <- msg
								break
							}
							//_, err = f.Write(message.Image)
							message.ImageUrl = fileDBurl
						}
					}
					rows, err := c.server.db.Query(`update room set description=$1, date=$2 where room=$3`,
						message.Body,
						c.room.Time,
						c.room.Id,
					)
					rows.Close()
					if err != nil {
						msg := ResponseMessage{Action: actionSendFirstMessage, Status: "db error", Code: 502}
						c.ch <- msg
					}
					c.room.channelForStatus <- roomNew
					c.room.channelForMessage <- message
				} else {
					msg := ResponseMessage{Action: actionSendFirstMessage, Status: "Room not found", Code: 404}
					c.ch <- msg
				}

			//закрытие комнаты
			case actionCloseRoom:
				log.Println(actionCloseRoom)
				c.room.Status = roomClose
				c.room.channelForStatus <- roomClose

			//
			case actionSendNickname:
				log.Println(actionSendNickname)
				var nickname ClientNickname
				err := json.Unmarshal(msg.Body, &nickname)
				if !CheckError(err, "Invalid RawData"+string(msg.Body), false) {
					msg := ResponseMessage{Action: actionSendNickname, Status: "Invalid Request", Code: 400}
					c.ch <- msg
				} else {
					c.Nick = nickname.Nickname
					rows, err := c.server.db.Query(`update room set nickname=$1 where room=$2`,
						nickname.Nickname,
						c.room.Id,
					)
					rows.Close()
					if err != nil {
						panic(err)
					} else {
						js, _ := json.Marshal(nickname)
						msg := ResponseMessage{Action: actionSendNickname, Status: "OK", Code: 200, Body: js}
						c.ch <- msg
						nickname.Rid = c.room.Id
						jsonstring, _ := json.Marshal(nickname)
						msg.Action = "updateInfo"
						msg.Body = jsonstring
						c.room.server.broadcast(msg)
					}
				}

				//
			case actionGetNickname:
				log.Println(actionGetNickname)
				var nickname sql.NullString
				log.Println(c.room.Id)
				_ = c.server.db.QueryRow("SELECT nickname FROM room WHERE room=?", c.room.Id).Scan(&nickname)
				log.Println(nickname)
				var n ClientNickname
				n.Nickname = c.Nick
				js, _ := json.Marshal(n)
				msg := ResponseMessage{Action: actionGetNickname, Status: "OK", Code: 200, Body: js}
				c.ch <- msg
				//}

			//получение всех сообщений
			case actionGetAllMessages:
				// log.Println(actionGetAllMessages)
				// messages, _ := json.Marshal(c.room.Messages)
				// response := ResponseMessage{Action: actionGetAllMessages, Status: "OK", Code: 200, Body: messages}
				// log.Println(response)
				messages := make([]Message, 0)
				rows, err := c.server.db.Query("SELECT room, type, date, body, url FROM message where room=$1", c.room.Id)
				if err != nil {
					rows.Close()
					msg := ResponseMessage{Action: actionGetAllMessages, Status: "Room not found", Code: 404, Body: msg.Body}
					c.ch <- msg
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
						rows, err := c.server.db.Query("SELECT room, type, date, body, url FROM message where room=$1", r.Id)
						if err != nil {
							rows.Close()
							msg := ResponseMessage{Action: actionGetAllMessages, Status: "Room not found", Code: 404, Body: msg.Body}
							c.ch <- msg
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

func DiffHandler(response http.ResponseWriter, request *http.Request) {
	id := request.URL.Query().Get("id")
	size := request.URL.Query().Get("size")

	if len(id) < 1 || len(size) < 1 {
		response.Header().Set("Access-Control-Allow-Origin", "*")
		response.WriteHeader(http.StatusBadRequest)
		response.Write([]byte("missing params"))
		return
	}
	s, _ := strconv.Atoi(size)

	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", db.DB_USER, db.DB_PASSWORD, db.DB_NAME)
	db, _ := sql.Open("postgres", dbinfo)

	messages := make([]Message, 0)
	var dbSize int
	db.QueryRow("SELECT count(*) FROM message where room=$1", id).Scan(&dbSize)
	if dbSize == s {
		response.Header().Set("Access-Control-Allow-Origin", "*")
		response.WriteHeader(http.StatusOK)
	} else {

		diff := dbSize - s
		if diff < 0 {
			response.Header().Set("Access-Control-Allow-Origin", "*")
			response.WriteHeader(http.StatusOK)
			return
		}
		rows, err := db.Query("SELECT room, type, date, body, url FROM message where room=$1 order by date desc limit $2", id, diff)
		if err != nil {
			rows.Close()
			response.WriteHeader(http.StatusInternalServerError)
			response.Write([]byte(err.Error()))
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
			jsonMessages, _ := json.Marshal(messages)
			msg := ResponseMessage{Action: "getDiff", Status: "OK", Code: 200, Body: jsonMessages}
			js, _ := json.Marshal(msg)
			response.Header().Set("Access-Control-Allow-Origin", "*")
			response.WriteHeader(http.StatusOK)
			response.Write(js)
		}
		//c.ch <-
	}
	//response.WriteHeader(http.StatusOK)
	// response.Write(js)
}
