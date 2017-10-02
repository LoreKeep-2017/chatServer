package chat

import "strconv"

type Message struct {
	Author string `json:"author"`
	Body   string `json:"body"`
	Room   int    `json:"room,omitempty"`
}

func (self *Message) String() string {
	return strconv.Itoa(self.Room) + ") " + self.Author + ": " + self.Body
}
