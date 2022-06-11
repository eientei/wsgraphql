package wsgraphql

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

type websocketRequest struct {
	ctx        mutable.Context
	outgoing   chan *apollows.Message
	operations map[string]mutable.Context
	ws         Conn
	server     *serverImpl
	wg         sync.WaitGroup
	m          sync.RWMutex
}

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

	var tickerch <-chan time.Time

	if server.keepalive > 0 {
		ticker := time.NewTicker(server.keepalive)

		defer func() {
			ticker.Stop()
		}()

		tickerch = ticker.C
	}

	req := &websocketRequest{
		ctx:        reqctx,
		outgoing:   make(chan *apollows.Message),
		operations: make(map[string]mutable.Context),
		ws:         ws,
		server:     server,
	}

	go req.readWebsocket()

	for {
		select {
		case <-reqctx.Done():
		case msg, ok := <-req.outgoing:
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

func (req *websocketRequest) writeWebsocketData(id string, data *graphql.Result) {
	t := apollows.OperationData
	if data.HasErrors() {
		t = apollows.OperationError
	}

	req.outgoing <- &apollows.Message{
		ID:   id,
		Type: t,
		Payload: apollows.Data{
			Value: data,
		},
	}
}

func (req *websocketRequest) writeWebsocketMessage(id string, t apollows.Operation, data interface{}) {
	req.outgoing <- &apollows.Message{
		ID:   id,
		Type: t,
		Payload: apollows.Data{
			Value: data,
		},
	}
}

func (req *websocketRequest) readWebsocketInit(msg *apollows.Message) (err error) {
	init := make(apollows.PayloadInit)

	err = json.Unmarshal(msg.Payload.RawMessage, &init)
	if err != nil {
		return
	}

	err = req.server.callbacks.OnConnect(req.ctx, init)
	if err != nil {
		return
	}

	req.writeWebsocketMessage("", apollows.OperationConnectionAck, nil)

	return
}

func (req *websocketRequest) readWebsocketStart(msg *apollows.Message) {
	req.m.RLock()
	prev, ok := req.operations[msg.ID]
	req.m.RUnlock()

	if ok {
		prev.Cancel()
	}

	opctx := mutable.NewMutableContext(req.ctx)

	req.m.Lock()
	req.operations[msg.ID] = opctx
	req.m.Unlock()

	req.wg.Add(1)

	go func() {
		payload, err := req.serveWebsocketOperation(opctx, msg)

		err = req.server.callbacks.OnOperationDone(opctx, &payload, err)

		if err != nil {
			req.writeWebsocketData(msg.ID, &graphql.Result{
				Errors: []gqlerrors.FormattedError{
					{
						Message: err.Error(),
					},
				},
			})
		}

		req.writeWebsocketMessage(msg.ID, apollows.OperationComplete, nil)

		opctx.Cancel()

		req.wg.Done()

		req.m.Lock()
		delete(req.operations, msg.ID)
		req.m.Unlock()
	}()
}

func (req *websocketRequest) readWebsocketStop(msg *apollows.Message) {
	req.m.RLock()

	prev, ok := req.operations[msg.ID]

	req.m.RUnlock()

	if ok {
		prev.Set(ContextKeyOperationStopped, true)
		prev.Cancel()
	}
}

func (req *websocketRequest) readWebsocketTerminate() {
	req.ctx.Set(ContextKeyOperationStopped, true)

	req.ctx.Cancel()
}

func (req *websocketRequest) readWebsocket() {
	var err error

	defer func() {
		if err != nil {
			req.writeWebsocketMessage("", apollows.OperationConnectionError, err.Error())
		}

		req.ctx.Cancel()

		req.wg.Wait()

		close(req.outgoing)
	}()

	for {
		var msg apollows.Message

		err = req.ws.ReadJSON(&msg)
		if err != nil {
			return
		}

		switch msg.Type {
		case apollows.OperationConnectionInit:
			err = req.readWebsocketInit(&msg)
		case apollows.OperationStart:
			req.readWebsocketStart(&msg)
		case apollows.OperationStop:
			req.readWebsocketStop(&msg)
		case apollows.OperationTerminate:
			req.readWebsocketTerminate()
		}

		if err != nil {
			return
		}
	}
}

func (req *websocketRequest) serveWebsocketOperation(
	opctx mutable.Context,
	msg *apollows.Message,
) (payload apollows.PayloadOperation, err error) {
	err = json.Unmarshal(msg.Payload.RawMessage, &payload)
	if err != nil {
		return
	}

	err = req.server.callbacks.OnConnect(opctx, nil)
	if err != nil {
		return
	}

	params, astdoc, subscription, result := req.server.parseAST(opctx, &payload)
	if result != nil {
		err = req.server.callbacks.OnOperation(opctx, &payload)
		if err != nil {
			return
		}

		req.writeWebsocketData(msg.ID, result)

		return
	}

	err = req.server.callbacks.OnOperation(opctx, &payload)
	if err != nil {
		return
	}

	var cres chan *graphql.Result

	if subscription {
		cres = graphql.ExecuteSubscription(graphql.ExecuteParams{
			Schema:        req.server.schema,
			Root:          req.server.rootObject,
			AST:           astdoc,
			OperationName: payload.OperationName,
			Args:          payload.Variables,
			Context:       params.Context,
		})
	} else {
		cres = make(chan *graphql.Result, 1)
		cres <- graphql.Execute(graphql.ExecuteParams{
			Schema:        req.server.schema,
			Root:          req.server.rootObject,
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

			err = req.server.callbacks.OnOperationResult(opctx, &payload, result)
			if err != nil {
				return
			}

			req.writeWebsocketData(msg.ID, result)
		}

		if result.HasErrors() {
			return
		}
	}
}
