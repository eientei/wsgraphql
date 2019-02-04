// Implemenentation of GraphQL over WebSocket Protocol by apollographql
// https://github.com/apollographql/subscriptions-transport-ws/blob/master/PROTOCOL.md
package proto

import "encoding/json"

// OperationMessage type
type OperationType string

const (
	GQLUnknown OperationType = ""

	// Pseudo types, used in events to signal state changes

	GQLConnectionClose = "connection_close"

	// Client to Server types

	GQLConnectionInit      = "connection_init"
	GQLStart               = "start"
	GQLStop                = "stop"
	GQLConnectionTerminate = "connection_terminate"

	// Server to Client

	GQLConnectionError     = "connection_error"
	GQLConnectionAck       = "connection_ack"
	GQLData                = "data"
	GQLError               = "error"
	GQLComplete            = "complete"
	GQLConnectionKeepAlive = "ka"
)

// Creates new operation message
func NewMessage(id interface{}, payload interface{}, t OperationType) *Message {
	return &Message{
		Id:      id,
		Payload: &PayloadBytes{Value: payload},
		Type:    t,
	}
}

// Generic Message type
type Message struct {
	Id      interface{}   `json:"id"`
	Payload *PayloadBytes `json:"payload"`
	Type    OperationType `json:"type"`
}

// PayloadBytes for operation parametrization
type PayloadOperation struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
}

// PayloadBytes for result of operation execution
type PayloadData struct {
	Data   interface{} `json:"data"`
	Errors []error     `json:"errors"`
}

// Utility combining functionality of interface{} and json.RawMessage
type PayloadBytes struct {
	Bytes []byte
	Value interface{}
}

// Save bytes for later deserialization, just as json.RawMessage
func (payload *PayloadBytes) UnmarshalJSON(b []byte) error {
	payload.Bytes = b
	return nil
}

// Serialize interface value
func (payload *PayloadBytes) MarshalJSON() ([]byte, error) {
	return json.Marshal(payload.Value)
}
