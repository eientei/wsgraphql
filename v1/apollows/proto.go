// Package apollows provides implementation of GraphQL over WebSocket Protocol as defined by apollo graphql
// see https://github.com/apollographql/subscriptions-transport-ws/blob/master/PROTOCOL.md for reference
package apollows

import (
	"encoding/json"
	"io"
)

// WebsocketSubprotocol defines websocket subprotocol expected by v1.apollows implementations
const WebsocketSubprotocol = "graphql-ws"

// Operation type is used to enumerate possible apollo message types
type Operation string

const (
	// OperationConnectionInit is set by the connecting client to initialize the websocket state with connection params
	// (if any)
	OperationConnectionInit Operation = "connection_init"

	// OperationStart client request initiates new operation
	// each operation may have 1-N OperationData / OperationError responses before being OperationComplete by the server
	// (potentially in response to OperationStop client request)
	OperationStart Operation = "start"

	// OperationStop client request to stop previously initiated operation with OperationStart
	OperationStop Operation = "stop"

	// OperationTerminate client request to gracefully close the connection
	OperationTerminate Operation = "connection_terminate"

	// OperationConnectionError server response to unsuccessful OperationConnectionInit attempt
	OperationConnectionError Operation = "connection_error"

	// OperationConnectionAck server response to successful OperationConnectionInit attempt
	OperationConnectionAck Operation = "connection_ack"

	// OperationData server response to previously initiated operation with OperationStart
	// may be multiple within same operation, specifically for subscriptions
	OperationData Operation = "data"

	// OperationError server response to previously initiated operation with OperationStart
	// may be multiple within same operation, specifically for subscriptions
	OperationError Operation = "error"

	// OperationComplete server response to terminate previously initiated operation with OperationStart
	// should be preceded by at least one OperationData or OperationError
	OperationComplete Operation = "complete"

	// OperationKeepAlive server response sent periodically to maintain websocket connection open
	OperationKeepAlive Operation = "ka"
)

// Data encapsulates both client and server json payload, combining json.RawMessage for decoding and
// arbitrary interface{} type for encoding
type Data struct {
	Value interface{}
	json.RawMessage
}

// ReadPayloadData client-side method to parse server response
func (payload *Data) ReadPayloadData() (*PayloadDataResponse, error) {
	if payload == nil {
		return nil, io.ErrUnexpectedEOF
	}

	var pd PayloadDataResponse

	err := json.Unmarshal(payload.RawMessage, &pd)
	if err != nil {
		return nil, err
	}

	payload.Value = pd

	return &pd, nil
}

// UnmarshalJSON stores provided json as a RawMessage, as well as initializing Value to same RawMessage
// to support both identity re-serialization and modification of Value after initialization
func (payload *Data) UnmarshalJSON(bs []byte) (err error) {
	payload.RawMessage = bs
	payload.Value = payload.RawMessage

	return nil
}

// MarshalJSON marshals either provided or deserialized Value as json
func (payload Data) MarshalJSON() (bs []byte, err error) {
	return json.Marshal(payload.Value)
}

// PayloadInit provides connection params
type PayloadInit map[string]interface{}

// PayloadOperation provides description for client-side operation initiation
type PayloadOperation struct {
	Variables     map[string]interface{} `json:"variables"`
	Extensions    map[string]interface{} `json:"extensions"`
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
}

// PayloadData provides server-side response for previously started operation
type PayloadData struct {
	Data Data `json:"data,omitempty"`

	// see https://github.com/graph-gophers/graphql-go#custom-errors
	// for adding custom error attributes
	Errors []error `json:"errors,omitempty"`
}

// PayloadErrorLocation error location in originating request
type PayloadErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// PayloadError client-side error representation
type PayloadError struct {
	Extensions map[string]interface{} `json:"extensions"`
	Message    string                 `json:"message"`
	Locations  []PayloadErrorLocation `json:"locations"`
	Path       []string               `json:"path"`
}

// PayloadDataResponse provides client-side payload representation
type PayloadDataResponse struct {
	Data   map[string]interface{} `json:"data,omitempty"`
	Errors []PayloadError         `json:"errors,omitempty"`
}

// Message encapsulates every message within v1.apollows protocol in both directions
type Message struct {
	ID      string    `json:"id,omitempty"`
	Type    Operation `json:"type"`
	Payload Data      `json:"payload,omitempty"`
}
