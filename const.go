package minevents

import (
	"context"
	"encoding/json"
)

type EventType string


type Event struct {
	ID   string `json:"id"`
	Type EventType
	Ctx  context.Context `json:"ctx,omitempty"`
	Dict map[string]interface{}
}

func (e *Event) String() string {
	b, _ := json.Marshal(e)
	return string(b)
}
