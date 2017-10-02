package chat

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"golang.org/x/net/websocket"
)

// Chat server.
type Server struct {
	pattern string
	//сообщения
	messages []*Message
	//типы пользователей
	clients   map[int]*Client
	operators map[int]*Operator
	rooms     map[*Client]*Room
	//операции
	//клиент
	addCh chan *Client
	delCh chan *Client
	//оператор
	addOCh chan *Operator
	delOCh chan *Operator
	//комнаты
	addRoomCh chan map[*Client]*Operator
	delRoomCh chan *Client
	//остальное
	sendAllCh chan *Message
	doneCh    chan bool
	errCh     chan error
}

// Create new chat server.
func NewServer(pattern string) *Server {
	messages := []*Message{}
	clients := make(map[int]*Client)
	operators := make(map[int]*Operator)
	rooms := make(map[*Client]*Room)
	addCh := make(chan *Client)
	delCh := make(chan *Client)
	addOCh := make(chan *Operator)
	delOCh := make(chan *Operator)
	addRoomCh := make(chan map[*Client]*Operator)
	delRoomCh := make(chan *Client)
	sendAllCh := make(chan *Message)
	doneCh := make(chan bool)
	errCh := make(chan error)

	return &Server{
		pattern,
		messages,
		clients,
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

func (s *Server) CreateRoom(cid string, operator *Operator) {
	id, err := strconv.Atoi(cid)
	if err != nil {
		// handle error
		log.Println(err)
		os.Exit(2)
	}
	if client, ok := s.clients[id]; ok {
		//do something here
		room := make(map[*Client]*Operator)
		s.delCh <- client
		room[client] = operator
		s.addRoomCh <- room
	}

}

func (s *Server) SendAll(msg *Message) {
	s.sendAllCh <- msg
}

func (s *Server) Done() {
	s.doneCh <- true
}

func (s *Server) Err(err error) {
	s.errCh <- err
}

func (s *Server) sendPastMessages(c *Client) {
	for _, msg := range s.messages {
		c.Write(msg)
	}
}

func (s *Server) sendAllClients(o *Operator) {
	o.sendAllClients(s.clients)
}

func (s *Server) sendAll(msg *Message) {
	for _, c := range s.clients {
		c.Write(msg)
	}
}

func (s *Server) broadcastFreeClients() {
	for _, operator := range s.operators {
		log.Println("sendAllfreeClinets")
		operator.sendAllClients(s.clients)
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

		client := NewClient(ws, s, "nick")
		s.Add(client)
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
	http.Handle(s.pattern, websocket.Handler(onConnected))
	http.Handle("/operator", websocket.Handler(onConnectedOperator))
	log.Println("Created handlers")

	for {
		select {

		// Add new a client
		case c := <-s.addCh:
			log.Println("Added new client")
			s.clients[c.Id] = c
			log.Println("Now", len(s.clients), "clients connected.")
			s.broadcastFreeClients()

		// del a client
		case c := <-s.delCh:
			log.Println("Delete client")
			delete(s.clients, c.Id)
			s.broadcastFreeClients()

		// Add new a operator
		case o := <-s.addOCh:
			log.Println("Added new operator")
			s.operators[o.id] = o
			log.Println("Now", len(s.operators), "operators connected.")
			s.sendAllClients(o)

		// del a operator
		case o := <-s.delOCh:
			log.Println("Delete operator")
			delete(s.operators, o.id)

		//Create room
		case clientOperator := <-s.addRoomCh:
			for client, operator := range clientOperator {
				log.Println("Creating new room")
				room := NewRoom(client, operator)
				operator.addToRoomCh <- room
				client.addRoomCh <- room
				log.Println(room)
				s.rooms[client] = room
				room.Listen()
			}

		// broadcast message for all clients
		// case msg := <-s.sendAllCh:
		// 	log.Println("Send all:", msg)
		// 	s.messages = append(s.messages, msg)
		// 	s.sendAll(msg)

		case err := <-s.errCh:
			log.Println("Error:", err.Error())

		case <-s.doneCh:
			return
		}
	}
}
