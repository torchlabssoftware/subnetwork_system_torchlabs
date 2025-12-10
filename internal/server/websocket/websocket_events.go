package server

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type EventHandler func(event Event, w *Worker) error

type loginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type successPayload struct {
	Success bool        `json:"success"`
	Payload interface{} `json:"payload"`
}

type errorPayload struct {
	Success bool        `json:"success"`
	Payload interface{} `json:"payload"`
}
