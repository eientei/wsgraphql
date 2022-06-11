package wsgraphql

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/graphql-go/graphql"
)

func (server *serverImpl) serveWebsocketRequest(
	reqctx mutable.Context,
	w http.ResponseWriter,
	r *http.Request,
) (err error) {
	ws, err := server.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	reqctx.Set(ContextKeyWebsocketConnection, ws)

	outgoing := make(chan *apollows.Message)

	var tickerch <-chan time.Time

	if server.keepalive > 0 {
		ticker := time.NewTicker(server.keepalive)

		defer func() {
			ticker.Stop()
		}()

		tickerch = ticker.C
	}

	go server.readWebsocket(reqctx, ws, outgoing)

	for {
		select {
		case <-reqctx.Done():
		case msg, ok := <-outgoing:
			if !ok {
				return
			}

			err = ws.WriteJSON(msg)
			if err != nil {
				return
			}
		case <-tickerch:
			err = ws.WriteJSON(&apollows.Message{
				Type: apollows.OperationKeepAlive,
			})
			if err != nil {
				return
			}
		}
	}
}

func (server *serverImpl) writeWebsocketData(ws Conn, id string, t apollows.Operation, data interface{}) {
	_ = ws.WriteJSON(&apollows.Message{
		ID:   id,
		Type: t,
		Payload: apollows.Data{
			Value: data,
		},
	})
}

func (server *serverImpl) readWebsocket(reqctx mutable.Context, ws Conn, outgoing chan *apollows.Message) {
	var err error

	wg := &sync.WaitGroup{}

	defer func() {
		if err != nil {
			server.writeWebsocketData(ws, "", apollows.OperationConnectionError, err.Error())
		}

		reqctx.Cancel()

		wg.Wait()

		close(outgoing)
	}()

	operations := make(map[string]mutable.Context)

	m := &sync.RWMutex{}

	for {
		var msg apollows.Message

		err = ws.ReadJSON(&msg)

		if err != nil {
			return
		}

		switch msg.Type {
		case apollows.OperationConnectionInit:
			init := make(apollows.PayloadInit)

			err = json.Unmarshal(msg.Payload.RawMessage, &init)
			if err != nil {
				return
			}

			err = server.callbacks.OnConnect(reqctx, init)
			if err != nil {
				return
			}

			server.writeWebsocketData(ws, "", apollows.OperationConnectionAck, nil)
		case apollows.OperationStart:
			m.RLock()

			prev, ok := operations[msg.ID]

			m.RUnlock()

			if ok {
				prev.Cancel()
			}

			opctx := mutable.NewMutableContext(reqctx)

			m.Lock()

			operations[msg.ID] = opctx

			m.Unlock()

			wg.Add(1)

			go func() {
				server.serveWebsocketOperation(opctx, msg, outgoing)

				opctx.Cancel()

				wg.Done()

				m.Lock()

				delete(operations, msg.ID)

				m.Unlock()
			}()
		case apollows.OperationStop:
			m.RLock()

			prev, ok := operations[msg.ID]

			m.RUnlock()

			if ok {
				prev.Set(ContextKeyOperationStopped, true)
				prev.Cancel()
			}
		case apollows.OperationTerminate:
			reqctx.Set(ContextKeyOperationStopped, true)

			reqctx.Cancel()
		}
	}
}

func (server *serverImpl) serveWebsocketOperation(
	opctx mutable.Context,
	msg apollows.Message,
	outgoing chan *apollows.Message,
) {
	var err error

	var payload apollows.PayloadOperation

	defer func() {
		err = server.callbacks.OnOperationDone(opctx, &payload, err)

		if err != nil {
			outgoing <- &apollows.Message{
				ID:   msg.ID,
				Type: apollows.OperationError,
				Payload: apollows.Data{
					Value: apollows.PayloadData{
						Errors: []error{err},
					},
				},
			}
		}

		outgoing <- &apollows.Message{
			ID:   msg.ID,
			Type: apollows.OperationComplete,
		}
	}()

	err = json.Unmarshal(msg.Payload.RawMessage, &payload)
	if err != nil {
		return
	}

	err = server.callbacks.OnConnect(opctx, nil)
	if err != nil {
		return
	}

	params, astdoc, subscription, result := server.parseAST(opctx, &payload)
	if result != nil {
		err = server.callbacks.OnOperation(opctx, &payload)
		if err != nil {
			return
		}

		outgoing <- &apollows.Message{
			ID:   msg.ID,
			Type: apollows.OperationError,
			Payload: apollows.Data{
				Value: result,
			},
		}

		return
	}

	err = server.callbacks.OnOperation(opctx, &payload)
	if err != nil {
		return
	}

	var cres chan *graphql.Result

	if subscription {
		cres = graphql.ExecuteSubscription(graphql.ExecuteParams{
			Schema:        server.schema,
			Root:          server.rootObject,
			AST:           astdoc,
			OperationName: payload.OperationName,
			Args:          payload.Variables,
			Context:       params.Context,
		})
	} else {
		cres = make(chan *graphql.Result, 1)

		cres <- graphql.Execute(graphql.ExecuteParams{
			Schema:        server.schema,
			Root:          server.rootObject,
			AST:           astdoc,
			OperationName: payload.OperationName,
			Args:          payload.Variables,
			Context:       params.Context,
		})

		close(cres)
	}

	var ok bool

	for {
		select {
		case <-params.Context.Done():
			if !ContextOperationStopped(params.Context) {
				err = params.Context.Err()
			}

			return
		case result, ok = <-cres:
			if !ok {
				return
			}

			t := apollows.OperationData
			if result.HasErrors() {
				t = apollows.OperationError
			}

			err = server.callbacks.OnOperationResult(opctx, &payload, result)
			if err != nil {
				return
			}

			outgoing <- &apollows.Message{
				ID:   msg.ID,
				Type: t,
				Payload: apollows.Data{
					Value: result,
				},
			}
		}
	}
}
