package wsgraphql

import "net/http"

// Upgrader interface used to upgrade HTTP request/response pair into a Conn
type Upgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (Conn, error)
}

// Conn interface is used to abstract connection returned from Upgrader
type Conn interface {
	ReadJSON(v interface{}) error
	WriteJSON(v interface{}) error
	Close(code int, message string) error
}
