// Package gorillaws provides compatability for gorilla websocket upgrader
package gorillaws

import (
	"net/http"

	"github.com/eientei/wsgraphql"

	"github.com/gorilla/websocket"
)

// Wrapper for gorilla websocket upgrader
type Wrapper struct {
	*websocket.Upgrader
}

// Upgrade implementation
func (g Wrapper) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (wsgraphql.Conn, error) {
	return g.Upgrader.Upgrade(w, r, responseHeader)
}

// Wrap gorilla upgrader into wsgraphql-compatible interface
func Wrap(upgrader *websocket.Upgrader) Wrapper {
	return Wrapper{
		Upgrader: upgrader,
	}
}
