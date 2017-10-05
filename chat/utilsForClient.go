package chat

const (
	actionSendDescriptionRoom = "sendDescriptionRoom"
)

//// Client messages

type ClientSendMessageRequest struct {
	Msg string `json:"msg"`
}

type ClientSendDescriptionRoomRequest struct {
	Description string `json:"description"`
	Title       string `json:"title"`
}

type ClientGreetingResponse struct {
	Action string `json:"action"`
	Msg    string `json:"data"`
}
