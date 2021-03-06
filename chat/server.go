package chat

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"

	"github.com/LoreKeep-2017/chatServer/db"
	//"golang.org/x/net/websocket"
)

const (
	operatorHandlerPattern = "/api/v1/operator/"
	clientHandlerPattern   = "/api/v1/client/"
)

// Chat server.
type Server struct {
	Router *mux.Router
	//сообщения
	messages []*Message
	//типы пользователей
	operators map[int]*Operator
	rooms     map[int]*Room
	db        *sql.DB
	//операции
	//клиент
	addCh chan *Client
	delCh chan *Client
	//оператор
	addOCh chan *Operator
	delOCh chan *Operator
	//комнаты
	addRoomCh chan map[Client]Operator
	delRoomCh chan *Client
	//остальное
	sendAllCh chan *Message
	doneCh    chan bool
	errCh     chan error
}

// Create new chat server.
func NewServer() *Server {
	r := mux.NewRouter()
	messages := []*Message{}
	operators := make(map[int]*Operator)
	rooms := make(map[int]*Room)
	dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", db.DB_USER, db.DB_PASSWORD, db.DB_NAME)
	db, err := sql.Open("postgres", dbinfo)
	addCh := make(chan *Client)
	delCh := make(chan *Client)
	addOCh := make(chan *Operator)
	delOCh := make(chan *Operator)
	addRoomCh := make(chan map[Client]Operator)
	delRoomCh := make(chan *Client)
	sendAllCh := make(chan *Message)
	doneCh := make(chan bool)
	errCh := make(chan error)

	if err != nil {
		panic(err.Error())
	}

	return &Server{
		r,
		messages,
		operators,
		rooms,
		db,
		addCh,
		delCh,
		addOCh,
		delOCh,
		addRoomCh,
		delRoomCh,
		sendAllCh,
		doneCh,
		errCh,
	}
}

func (s *Server) Add(c *Client) {
	s.addCh <- c
}

func (s *Server) AddOperator(o *Operator) {
	s.addOCh <- o
}

func (s *Server) Del(c *Client) {
	log.Println("delete", c)
	s.delCh <- c
}

func (s *Server) DelOperator(o *Operator) {
	s.delOCh <- o
}

func (s *Server) Done() {
	s.doneCh <- true
}

func (s *Server) Err(err error) {
	s.errCh <- err
}

func (s *Server) broadcast(responseMessage ResponseMessage) {
	for _, operator := range s.operators {
		operator.ch <- responseMessage
	}
}

func (s *Server) createResponseAllRooms() ResponseMessage {
	response := OperatorResponseRooms{s.rooms, len(s.rooms)}
	jsonstring, _ := json.Marshal(response)
	msg := ResponseMessage{Action: actionGetAllRooms, Status: "OK", Code: 200, Body: jsonstring}
	return msg
}

func (s *Server) sendMessageToOperator(id int, action string, jsonstring []byte) {
	//response := OperatorResponseRooms{s.rooms, len(s.rooms)}
	//jsonstring, _ := json.Marshal(response)
	operator := s.operators[id]
	msg := ResponseMessage{Action: action, Status: "OK", Code: 200, Body: jsonstring}
	operator.ch <- msg
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Listen and serve.
// It serves client connection and broadcast request.
func (s *Server) Listen() {

	log.Println("Listening server...")

	// websocket handler for client
	onConnected := func(w http.ResponseWriter, r *http.Request) {

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			//return
		}
		defer func() {
			err := conn.Close()
			if err != nil {
				s.errCh <- err
			}
		}()

		room := NewRoom(s)
		client := NewClient(conn, s, room)
		room.Client = client
		s.rooms[room.Id] = room
		s.Add(client)
		room.Listen()
		client.Listen()
	}

	// websocket handler for operator
	onConnectedOperator := func(w http.ResponseWriter, r *http.Request) {

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			// return
		}
		defer func() {
			err := conn.Close()
			if err != nil {
				s.errCh <- err
			}
		}()
		operator := NewOperator(conn, s)
		s.AddOperator(operator)
		operator.Listen()
	}
	// websocket.
	// 	http.HandleF
	s.Router.HandleFunc(clientHandlerPattern, onConnected)
	s.Router.HandleFunc(operatorHandlerPattern, onConnectedOperator)
	//http.Handle(clientHandlerPattern, onConnected)
	//http.Handle(operatorHandlerPattern, websocket.Handler(onConnectedOperator))
	log.Println("Created handlers")

	for {
		select {

		// Add new a client
		case <-s.addCh:
			//msg := s.createResponseAllRooms()
			//s.broadcast(msg)

			// del a client
		case c := <-s.delCh:

			log.Println("Delete client", c.room)
			// c.room.Status = roomClose
			// c.room.channelForStatus <- roomClose
			// if c.room.Operator != nil {
			// 	log.Println("rooms", c.room.Operator.rooms)
			// 	delete(c.room.Operator.rooms, c.room.Id)
			// }

		// Add new a operator
		case o := <-s.addOCh:
			log.Println("Added new operator")
			s.operators[o.Id] = o

		// del a operator
		case o := <-s.delOCh:
			log.Println("Delete operator")
			delete(s.operators, o.Id)

		case err := <-s.errCh:
			log.Println("Error:", err.Error())

		case <-s.doneCh:
			return
		}
	}
}
