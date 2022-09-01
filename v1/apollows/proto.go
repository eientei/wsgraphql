// Package apollows provides implementation of GraphQL over WebSocket Protocol as defined by
// https://github.com/apollographql/subscriptions-transport-ws/blob/master/PROTOCOL.md  [GWS]
// https://github.com/enisdenjo/graphql-ws/blob/master/PROTOCOL.md [GTWS]
package apollows

import (
	"encoding/json"
	"io"
)

// Protocol websocket subprotocol defining server behavior
type Protocol string

const (
	// WebsocketSubprotocolGraphqlWS websocket subprotocol expected by subscriptions-transport-ws implementations
	WebsocketSubprotocolGraphqlWS Protocol = "graphql-ws"

	// WebsocketSubprotocolGraphqlTransportWS websocket subprotocol exepected by graphql-ws implementations
	WebsocketSubprotocolGraphqlTransportWS Protocol = "graphql-transport-ws"
)

// Operation type is used to enumerate possible apollo message types
type Operation string

const (
	// OperationConnectionInit [GWS,GTWS]
	// is set by the connecting client to initialize the websocket state with connection params (if any)
	OperationConnectionInit Operation = "connection_init"

	// OperationStart [GWS]
	// client request initiates new operation, each operation may have 0-N OperationData responses before being
	// terminated by either OperationComplete or OperationError
	OperationStart Operation = "start"

	// OperationSubscribe [GTWS]
	// client request initiates new operation, each operation may have 0-N OperationNext responses before being
	// terminated by either OperationComplete or OperationError
	OperationSubscribe Operation = "subscribe"

	// OperationTerminate [GWS]
	// client request to gracefully close the connection, equialent to closing the websocket
	OperationTerminate Operation = "connection_terminate"

	// OperationConnectionError [GWS]
	// server response to unsuccessful OperationConnectionInit attempt
	OperationConnectionError Operation = "connection_error"

	// OperationConnectionAck [GWS,GTWS]
	// server response to successful OperationConnectionInit attempt
	OperationConnectionAck Operation = "connection_ack"

	// OperationData [GWS]
	// server response to previously initiated operation with OperationStart may be multiple within same operation,
	// specifically for subscriptions
	OperationData Operation = "data"

	// OperationNext [GTWS]
	// server response to previously initiated operation with OperationSubscribe may be multiple within same operation,
	// specifically for subscriptions
	OperationNext Operation = "next"

	// OperationError [GWS,GTWS]
	// server response to previously initiated operation with OperationStart/OperationSubscribe
	OperationError Operation = "error"

	// OperationStop [GWS]
	// client request to stop previously initiated operation with OperationStart
	OperationStop Operation = "stop"

	// OperationComplete [GWS,GTWS]
	// GWS: server response indicating previously initiated operation is complete
	// GTWS: server response indicating previously initiated operation is complete
	// GTWS: client request to stop previously initiated operation with OperationSubscribe
	OperationComplete Operation = "complete"

	// OperationKeepAlive [GWS]
	// server response sent periodically to maintain websocket connection open
	OperationKeepAlive Operation = "ka"

	// OperationPing [GTWS]
	// sever/client request for OperationPong response
	OperationPing Operation = "ping"

	// OperationPong [GTWS]
	// sever/client response for OperationPing request
	// can be sent at any time (without prior OperationPing) to maintain wesocket connection
	OperationPong Operation = "pong"
)

// Error providing MessageType to close websocket with
type Error interface {
	error
	EventMessageType() MessageType
}

type errorImpl struct {
	error
	message     string
	messageType MessageType
}

func (e errorImpl) EventMessageType() MessageType {
	return e.messageType
}

func (e errorImpl) Unwrap() error {
	return e.error
}

func (e errorImpl) Error() string {
	return e.message
}

// WrapError wraps provided error into Error
func WrapError(err error, messageType MessageType) Error {
	message := err.Error()

	if messageType.Error() != "" {
		message = messageType.Error() + ": " + message
	}

	return errorImpl{
		error:       err,
		message:     message,
		messageType: messageType,
	}
}

// NewSubscriberAlreadyExistsError constructs new Error using subscriber id as part of the message
func NewSubscriberAlreadyExistsError(id string) Error {
	return errorImpl{
		error:       nil,
		message:     "Subscriber for " + id + " already exists",
		messageType: EventSubscriberAlreadyExists,
	}
}

// MessageType websocket message types / status codes used to indicate protocol-level events following closing the
// websocket
type MessageType int

const (
	// EventCloseNormal standard websocket message type
	EventCloseNormal MessageType = 1000

	// EventCloseError standard websocket message type
	EventCloseError MessageType = 1006

	// EventInvalidMessage indicates invalid protocol message
	EventInvalidMessage MessageType = 4400

	// EventUnauthorized indicated attempt to subscribe to an operation before receiving OperationConnectionAck
	EventUnauthorized MessageType = 4401

	// EventInitializationTimeout indicates timeout occurring before client sending OperationConnectionInit
	EventInitializationTimeout MessageType = 4408

	// EventTooManyInitializationRequests indicates receiving more than one OperationConnectionInit
	EventTooManyInitializationRequests MessageType = 4429

	// EventSubscriberAlreadyExists indicates subscribed operation ID already being in use
	// (not yet terminated by either OperationComplete or OperationError)
	EventSubscriberAlreadyExists MessageType = 4409
)

var messageTypeDescriptions = map[MessageType]string{
	EventCloseNormal:                   "Termination requested",
	EventInvalidMessage:                "Invalid message",
	EventUnauthorized:                  "Unauthorized",
	EventInitializationTimeout:         "Connection initialisation timeout",
	EventTooManyInitializationRequests: "Too many initialisation requests",
}

// EventMessageType implementation
func (m MessageType) EventMessageType() MessageType {
	return m
}

// Error implementation
func (m MessageType) Error() string {
	return messageTypeDescriptions[m]
}

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

// ReadPayloadError client-side method to parse server error response
func (payload *Data) ReadPayloadError() (*PayloadError, error) {
	if payload == nil {
		return nil, io.ErrUnexpectedEOF
	}

	var pd PayloadError

	err := json.Unmarshal(payload.RawMessage, &pd)
	if err != nil {
		return nil, err
	}

	payload.Value = pd

	return &pd, nil
}

// ReadPayloadErrors client-side method to parse server error response
func (payload *Data) ReadPayloadErrors() (pds []*PayloadError, err error) {
	if payload == nil {
		return nil, io.ErrUnexpectedEOF
	}

	err = json.Unmarshal(payload.RawMessage, &pds)
	if err != nil {
		return nil, err
	}

	payload.Value = pds

	return pds, nil
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

// PayloadDataRaw provides server-side response for previously started operation
type PayloadDataRaw struct {
	Data Data `json:"data"`

	// see https://github.com/graph-gophers/graphql-go#custom-errors
	// for adding custom error attributes
	Errors []error `json:"errors,omitempty"`
}

// PayloadData type-alias for serialization
type PayloadData PayloadDataRaw

// MarshalJSON serializes PayloadData to JSON, excluding empty data
func (payload PayloadData) MarshalJSON() (bs []byte, err error) {
	var data *Data

	if payload.Data.Value != nil {
		data = &payload.Data
	}

	return json.Marshal(struct {
		Data *Data `json:"data,omitempty"`
		PayloadDataRaw
	}{
		PayloadDataRaw: PayloadDataRaw(payload),
		Data:           data,
	})
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

// MessageRaw encapsulates every message within apollows protocol in both directions
type MessageRaw struct {
	ID      string    `json:"id,omitempty"`
	Type    Operation `json:"type"`
	Payload Data      `json:"payload"`
}

// Message type-alias for (de-)serialization
type Message MessageRaw

// MarshalJSON serializes Message to JSON, excluding empty id or payload from serialized fields.
func (message Message) MarshalJSON() (bs []byte, err error) {
	var payload *Data

	if message.Payload.Value != nil {
		payload = &message.Payload
	}

	return json.Marshal(struct {
		Payload *Data `json:"payload,omitempty"`
		MessageRaw
	}{
		MessageRaw: MessageRaw(message),
		Payload:    payload,
	})
}
