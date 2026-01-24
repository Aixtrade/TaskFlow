package payload

type DemoPayload struct {
	Message string `json:"message"`
	Count   int    `json:"count,omitempty"`
}

type DemoResult struct {
	TaskID  string `json:"task_id"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}
