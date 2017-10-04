package chat

import "log"

// Operator methods
const (
	actionGetAllRooms = "getAllRooms"
	actionCreateRoom  = "createRoom"
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

type OperatorResponseAddToRoom struct {
	Room int `json:"roomID"`
}

type OperatorResponseRooms struct {
	Room []Room `json:"rooms"`
	Size int    `json:"size"`
}

type RequestCreateRoom struct {
	ID int `json:"cid"`
}

type OperatorSendMessage struct {
	Message string `json:"message"`
}
