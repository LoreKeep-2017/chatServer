package chat

import (
	"encoding/json"
	"log"
)

// Operator methods
const (
	actionGetAllClients = "getAllClients"
	actionCreateRoom    = "createRoom"
	actionDeleteRoom    = "deleteRoom"
)

// CheckError checks errors and print log
func CheckError(err error, message string, fatal bool) bool {
	if err != nil {
		if fatal {
			log.Fatalln(message + ": " + err.Error())
		} else {
			log.Println(message + ": " + err.Error())
		}
	}
	return err == nil
}

//// Operator messages

type OperatorRequest struct {
	Action  string          `json:"action"`
	RawData json.RawMessage `json:"data,omitempty"`
}

type OperatorResponseAddToRoom struct {
	Action string `json:"action"`
	Room   int    `json:"room"`
}

type OperatorGrabb struct {
	Id string `json:"cid"`
}

type OperatorSendMessage struct {
	Message string `json:"message"`
}
