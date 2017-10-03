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
	Author string `json:"author"`
	Body   string `json:"body"`
	Room   int    `json:"room,omitempty"`
}

//RequestMessage стандартное сообщение с фронтенда
type RequestMessage struct {
	Type   string          `json:"type"`
	Action string          `json:"action"`
	Body   json.RawMessage `json:"body,omitempty"`
}

func (self *Message) String() string {
	return strconv.Itoa(self.Room) + ") " + self.Author + ": " + self.Body
}
