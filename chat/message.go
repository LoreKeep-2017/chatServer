package chat

import (
	"encoding/json"
	"strconv"
)

const (
	actionSendMessage    = "sendMessage"
	actionGetAllMessages = "getAllMessages"
	actionCloseRoom      = "closeRoom"
)

type Message struct {
	Author string `json:"author,omitempty"`
	Body   string `json:"body"`
	Room   int    `json:"room,omitempty"`
	Time   int    `json:"time,omitempty"`
}

//RequestMessage стандартное сообщение с фронтенда
type RequestMessage struct {
	Type   string          `json:"type"`
	Action string          `json:"action"`
	Body   json.RawMessage `json:"body,omitempty"`
}

//ResponseMessage стандартное сообщение от сервера
type ResponseMessage struct {
	Action string          `json:"action"`
	Status string          `json:"status"`
	Code   int             `json:"code"`
	Body   json.RawMessage `json:"body,omitempty"`
}

func (self *Message) String() string {
	return strconv.Itoa(self.Room) + ") " + self.Author + ": " + self.Body
}
