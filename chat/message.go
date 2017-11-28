package chat

import (
	"encoding/json"
	"strconv"
)

const (
	actionSendMessage       = "sendMessage"
	actionGetAllMessages    = "getAllMessages"
	actionCloseRoom         = "closeRoom"
	actionChangeStatusRooms = "changeStatusRoom"
)

type Message struct {
	Author      string `json:"author,omitempty"`
	Body        string `json:"body,omitempty"`
	Room        int    `json:"room,omitempty"`
	Time        int    `json:"time,omitempty"`
	ImageUrl    string `json:"imageUrl,omitempty"`
	Image       string `json:"image,omitempty"`
	ImageFormat string `json:"imageFormat,omitempty"`
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
	Room   int             `json:"room,omitempty"`
	Body   json.RawMessage `json:"body,omitempty"`
}

func (self *Message) String() string {
	return strconv.Itoa(self.Room) + ") " + self.Author + ": " + self.Body
}

var FormatsImage = map[string]bool{
	"svg":  true,
	"png":  true,
	"jpg":  true,
	"jpeg": true,
	"gif":  true,
}
