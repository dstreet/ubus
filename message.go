package ubus

type Message struct {
	Event   string
	Data    any
	Headers map[string]any
}
