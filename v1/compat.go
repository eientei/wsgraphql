package wsgraphql

import "net/http"

// Upgrader interface used to upgrade HTTP request/response pair into a Conn
// signature based on github.com/gorilla/websocket.Upgrader, but decouples from specific implementation
type Upgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (Conn, error)
}

// Conn interface is used to abstract connection returned from Upgrader
// signature based on github.com/gorilla/websocket.Conn, but decouples from specific implementation
type Conn interface {
	ReadJSON(v interface{}) error
	WriteJSON(v interface{}) error
	Close() error
}
