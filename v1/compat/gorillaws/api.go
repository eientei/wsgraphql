// Package gorillaws provides compatibility for gorilla websocket upgrader
package gorillaws

import (
	"net/http"

	"github.com/eientei/wsgraphql/v1"
	"github.com/gorilla/websocket"
)

// Wrapper for gorilla websocket upgrader
type Wrapper struct {
	*websocket.Upgrader
}

type conn struct {
	*websocket.Conn
}

func (conn conn) Close(code int, message string) error {
	origerr := conn.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(code, message))

	err := conn.Conn.Close()
	if err == nil {
		err = origerr
	}

	return err
}

func (conn conn) Subprotocol() string {
	return conn.Conn.Subprotocol()
}

// Upgrade implementation
func (g Wrapper) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (wsgraphql.Conn, error) {
	c, err := g.Upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		return nil, err
	}

	return conn{
		Conn: c,
	}, nil
}

// Wrap gorilla upgrader into wsgraphql-compatible interface
func Wrap(upgrader *websocket.Upgrader) Wrapper {
	return Wrapper{
		Upgrader: upgrader,
	}
}
