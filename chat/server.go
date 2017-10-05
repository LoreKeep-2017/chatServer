package chat

import (
	"log"
	"net/http"

	"golang.org/x/net/websocket"
)

const (
	operatorHandlerPattern = "/api/v1/operator"
	clientHandlerPattern   = "/api/v1/client"
)

// Chat server.
type Server struct {
	//сообщения
	messages []*Message
	//типы пользователей
	operators map[int]*Operator
	rooms     map[int]*Room
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
	messages := []*Message{}
	operators := make(map[int]*Operator)
	rooms := make(map[int]*Room)
	addCh := make(chan *Client)
	delCh := make(chan *Client)
	addOCh := make(chan *Operator)
	delOCh := make(chan *Operator)
	addRoomCh := make(chan map[Client]Operator)
	delRoomCh := make(chan *Client)
	sendAllCh := make(chan *Message)
	doneCh := make(chan bool)
	errCh := make(chan error)

	return &Server{
		messages,
		operators,
		rooms,
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

func (s *Server) sendAllRooms(o *Operator) {
	o.sendAllRooms()
}

func (s *Server) broadcastRooms() {
	for _, operator := range s.operators {
		log.Println("sendAllRooms")
		operator.sendAllRooms()
	}
}

func (s *Server) broadcastChangeStatus(room Room) {
	for _, operator := range s.operators {
		log.Println("BroadcastChangestatus")
		operator.sendChangeStatus(room)
	}
}

// Listen and serve.
// It serves client connection and broadcast request.
func (s *Server) Listen() {

	log.Println("Listening server...")

	// websocket handler for client
	onConnected := func(ws *websocket.Conn) {
		defer func() {
			err := ws.Close()
			if err != nil {
				s.errCh <- err
			}
		}()

		room := NewRoom(s)
		client := NewClient(ws, s, "nick", room)
		room.Client = client
		s.rooms[room.Id] = room
		s.Add(client)
		room.Listen()
		client.Listen()
	}

	// websocket handler for operator
	onConnectedOperator := func(ws *websocket.Conn) {
		defer func() {
			err := ws.Close()
			if err != nil {
				s.errCh <- err
			}
		}()

		operator := NewOperator(ws, s)
		s.AddOperator(operator)
		operator.Listen()
	}
	http.Handle(clientHandlerPattern, websocket.Handler(onConnected))
	http.Handle(operatorHandlerPattern, websocket.Handler(onConnectedOperator))
	log.Println("Created handlers")

	for {
		select {

		// Add new a client
		case <-s.addCh:
			log.Println("Added new client")
			s.broadcastRooms()

		// del a client
		case <-s.delCh:
			log.Println("Delete client")
			s.broadcastRooms()

		// Add new a operator
		case o := <-s.addOCh:
			log.Println("Added new operator")
			s.operators[o.Id] = o
			log.Println("Now", len(s.operators), "operators connected.")
			s.sendAllRooms(o)

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
