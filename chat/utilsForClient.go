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
	Nick        string `json:"nick"`
}

type ClientGreetingResponse struct {
	Action string `json:"action"`
	Msg    string `json:"data"`
}
