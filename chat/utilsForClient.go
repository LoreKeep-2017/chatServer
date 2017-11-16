package chat

const (
	actionSendDescriptionRoom = "sendDescriptionRoom"
	actionSendFirstMessage    = "sendFirstMessage"
	actionRestoreRoom         = "restoreRoom"
	actionSendNickname        = "sendNickname"
	actionGetNickname         = "getNickname"
)

//// Client messages

type ClientSendMessageRequest struct {
	Msg string `json:"msg"`
}

type ClientRoom struct {
	RoomID int `json:"rid"`
}

type ClientGreetingResponse struct {
	Action string `json:"action"`
	Msg    string `json:"data"`
}

type ClientNickname struct {
	Nickname string `json:"nickname"`
	Rid      int    `json:"rid,omitempty"`
}
