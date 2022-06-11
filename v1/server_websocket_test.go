package wsgraphql

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestNewServerWebsocket(t *testing.T) {
	srv := testNewServer(t)

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{WebsocketSubprotocol},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "1",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `query { getFoo }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "1", msg.ID)
	assert.Equal(t, apollows.OperationData, msg.Type)

	pd, err := msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, 123, pd.Data["getFoo"])

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "1", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "2",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `mutation { setFoo(value: 3) }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "2", msg.ID)
	assert.Equal(t, apollows.OperationData, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, map[string]interface{}{
		"setFoo": true,
	}, pd.Data)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "2", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "3",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `mutation { setFoo }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "3", msg.ID)
	assert.Equal(t, apollows.OperationData, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, map[string]interface{}{
		"setFoo": false,
	}, pd.Data)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "3", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "4",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `mutation { bar }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "4", msg.ID)
	assert.Equal(t, apollows.OperationError, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Greater(t, len(pd.Errors), 0)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "4", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "5",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `subscription { forever }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.WriteJSON(apollows.Message{
		ID:   "5",
		Type: apollows.OperationStop,
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "5", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "6",
		Type: apollows.OperationStart,
		Payload: apollows.Data{
			Value: apollows.PayloadOperation{
				Query: `subscription { fooUpdates }`,
			},
		},
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "6", msg.ID)
	assert.Equal(t, apollows.OperationData, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, 1, pd.Data["fooUpdates"])

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "6", msg.ID)
	assert.Equal(t, apollows.OperationData, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, 2, pd.Data["fooUpdates"])

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "6", msg.ID)
	assert.Equal(t, apollows.OperationData, msg.Type)

	pd, err = msg.Payload.ReadPayloadData()

	assert.NoError(t, err)
	assert.Len(t, pd.Errors, 0)
	assert.EqualValues(t, 3, pd.Data["fooUpdates"])

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, "6", msg.ID)
	assert.Equal(t, apollows.OperationComplete, msg.Type)

	assert.NoError(t, conn.Close())
}

func TestNewServerWebsocketKeepalive(t *testing.T) {
	srv := testNewServer(t, WithKeepalive(time.Millisecond*10))

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{WebsocketSubprotocol},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationKeepAlive, msg.Type)
}

func TestNewServerWebsocketTerminate(t *testing.T) {
	srv := testNewServer(t)

	defer srv.Close()

	u := "ws" + strings.TrimPrefix(srv.URL, "http")

	conn, resp, err := websocket.DefaultDialer.Dial(u, http.Header{
		"sec-websocket-protocol": []string{WebsocketSubprotocol},
	})

	assert.NoError(t, err)

	defer func() {
		_ = conn.Close()
		_ = resp.Body.Close()
	}()

	err = conn.WriteJSON(apollows.Message{
		ID:      "",
		Type:    apollows.OperationConnectionInit,
		Payload: apollows.Data{},
	})

	assert.NoError(t, err)

	var msg apollows.Message

	err = conn.ReadJSON(&msg)

	assert.NoError(t, err)
	assert.Equal(t, apollows.OperationConnectionAck, msg.Type)

	err = conn.WriteJSON(apollows.Message{
		ID:   "",
		Type: apollows.OperationTerminate,
	})

	assert.NoError(t, err)

	err = conn.ReadJSON(&msg)

	assert.Error(t, err)
}
