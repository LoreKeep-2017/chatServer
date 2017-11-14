package chat

const (
	actionSendDescriptionRoom = "sendDescriptionRoom"
	actionSendFirstMessage    = "sendFirstMessage"
	actionRestoreRoom         = "restoreRoom"
)

//// Client messages

type ClientSendMessageRequest struct {
	Msg string `json:"msg"`
}

type ClientSendDescriptionRoomRequest struct {
	Description string `json:"description"`
	Title       string `json:"title"`
	Nick        string `json:"nick"`
}

type ClientRoom struct {
	RoomID int `json:"rid"`
}

type ClientGreetingResponse struct {
	Action string `json:"action"`
	Msg    string `json:"data"`
}
