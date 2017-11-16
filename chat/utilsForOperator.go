package chat

import "log"

// Operator methods
const (
	actionGetAllRooms      = "getAllRooms"
	actionEnterRoom        = "enterRoom"
	actionRoomStatusSend   = "roomStatusSend"
	actionGetRoomsByStatus = "getRoomsByStatus"
	actionGetOperators     = "getOperators"
	actionSendID           = "sendId"
	actionChangeOperator   = "changeOperator"
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
	Room map[int]*Room `json:"rooms"`
	Size int           `json:"size"`
}

type OperatorResponseRoomsNew struct {
	Room map[int]Room `json:"rooms"`
	Size int          `json:"size"`
}

type RequestActionWithRoom struct {
	ID int `json:"rid"`
}

type RequestTypeRooms struct {
	Type string `json:"type"`
}

type OperatorSendMessage struct {
	Message string `json:"message"`
}

type OperatorChange struct {
	ID   int `json:"to"`
	Room int `json:"rid"`
}
