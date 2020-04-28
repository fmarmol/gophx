package gophx

// Message ...
type Message struct {
	Topic   Topic       `json:"topic"`
	Event   Event       `json:"event"`
	Payload interface{} `json:"payload"`
	Ref     int         `json:"ref"`
}
