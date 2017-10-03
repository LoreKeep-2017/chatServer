package chat

//// Client messages

type ClientSendMessageRequest struct {
	Msg string `json:"msg"`
}

type ClientGreetingResponse struct {
	Action string `json:"action"`
	Msg    string `json:"data"`
}
