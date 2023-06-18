package wsgraphql

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

type websocketRequest struct {
	ctx        context.Context
	outgoing   chan outgoingMessage
	operations map[string]mutable.Context
	ws         Conn
	server     *serverImpl
	protocol   apollows.Protocol
	wg         sync.WaitGroup
	m          sync.RWMutex
	init       bool
}

type outgoingMessage struct {
	*apollows.Message
	apollows.Error
}

func (server *serverImpl) serveWebsocketRequest(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
) (err error) {
	ws, err := server.upgrader.Upgrade(w, r, w.Header())
	if err != nil {
		return
	}

	reqctx := RequestContext(ctx)

	reqctx.Set(ContextKeyWebsocketConnection, ws)
	reqctx.Set(ContextKeyHTTPResponseStarted, true)

	protocol := apollows.Protocol(ws.Subprotocol())

	_, known := server.subscriptionProtocols[protocol]
	if !known {
		if ws != nil {
			_ = ws.Close(int(apollows.EventCloseNormal), apollows.ErrUnknownProtocol.Error())
		}

		return apollows.ErrUnknownProtocol
	}

	req := &websocketRequest{
		protocol:   protocol,
		ctx:        ctx,
		outgoing:   make(chan outgoingMessage, 1),
		operations: make(map[string]mutable.Context),
		ws:         ws,
		server:     server,
	}

	var tickerType apollows.Operation

	switch req.protocol {
	case apollows.WebsocketSubprotocolGraphqlWS:
		tickerType = apollows.OperationKeepAlive
	case apollows.WebsocketSubprotocolGraphqlTransportWS:
		tickerType = apollows.OperationPong
	}

	var tickerch <-chan time.Time

	if server.keepalive > 0 {
		ticker := time.NewTicker(server.keepalive)

		defer func() {
			ticker.Stop()
		}()

		tickerch = ticker.C
	}

	go req.readWebsocket()

	// req.outgoing is read to completion to avoid any potential blocking
	// readWebsocket exit is ensured by closing a websocket on any error, this causes req.ws.ReadJSON() to return
	for {
		select {
		case msg, ok := <-req.outgoing:
			if !ok {
				return
			}

			switch {
			case msg.Message != nil:
				err = ws.WriteJSON(msg.Message)
			case msg.Error != nil:
				err = ws.Close(int(msg.Error.EventMessageType()), msg.Error.Error())
			}
		case <-tickerch:
			err = ws.WriteJSON(&apollows.Message{
				Type: tickerType,
			})
		}

		if err != nil {
			_ = ws.Close(int(apollows.EventCloseNormal), err.Error())
		}
	}
}

func combineErrors(errs []gqlerrors.FormattedError) gqlerrors.FormattedError {
	if len(errs) == 1 {
		return errs[0]
	}

	errmsg := "preparing operation"

	var errmsgs []string

	combinedext := make(map[string]interface{})

	for _, err := range errs {
		fmterr := FormatError(err)

		for k, v := range fmterr.Extensions {
			combinedext[k] = v
		}

		errmsgs = append(errmsgs, fmterr.Error())
	}

	if len(errmsgs) > 0 {
		errmsg += ": " + strings.Join(errmsgs, "; ")
	}

	rooterr := gqlerrors.NewFormattedError(errmsg)

	if len(errs) > 0 {
		combinedext["errors"] = errs
		rooterr.Extensions = combinedext
	}

	return rooterr
}

func (req *websocketRequest) handleError(ctx context.Context, err error) {
	awerr, ok := err.(apollows.Error)
	if ok {
		if req.protocol == apollows.WebsocketSubprotocolGraphqlWS {
			req.writeWebsocketMessage(
				ctx,
				apollows.OperationConnectionError,
				gqlerrors.FormatError(awerr),
			)
		}

		req.outgoing <- outgoingMessage{
			Error: awerr,
		}

		return
	}

	res, ok := err.(ResultError)

	if ok {
		switch {
		case req.protocol == apollows.WebsocketSubprotocolGraphqlWS:
			req.writeWebsocketMessage(ctx, apollows.OperationError, combineErrors(res.Result.Errors))
		default:
			req.writeWebsocketMessage(ctx, apollows.OperationError, res.Result.Errors)
		}

		return
	}

	req.writeWebsocketMessage(ctx, apollows.OperationError, gqlerrors.FormatError(err))
}

func (req *websocketRequest) writeWebsocketData(ctx context.Context, data *graphql.Result) {
	var t apollows.Operation

	switch req.protocol {
	case apollows.WebsocketSubprotocolGraphqlWS:
		t = apollows.OperationData
	case apollows.WebsocketSubprotocolGraphqlTransportWS:
		t = apollows.OperationNext

		if ContextOperationStopped(ctx) {
			return
		}
	}

	req.writeWebsocketMessage(ctx, t, data)
}

func (req *websocketRequest) writeWebsocketMessage(ctx context.Context, t apollows.Operation, data interface{}) {
	if t == apollows.OperationError {
		OperationContext(ctx).Set(ContextKeyOperationStopped, true)
	}

	select {
	case req.outgoing <- outgoingMessage{
		Message: &apollows.Message{
			ID:   ContextOperationID(ctx),
			Type: t,
			Payload: apollows.Data{
				Value: data,
			},
		},
	}:
	case <-RequestContext(ctx).Done():
	}
}

func (req *websocketRequest) readWebsocketInit(msg *apollows.Message) (err error) {
	init := make(apollows.PayloadInit)

	if len(msg.Payload.RawMessage) > 0 {
		err = json.Unmarshal(msg.Payload.RawMessage, &init)
		if err != nil {
			return
		}
	}

	err = req.server.interceptors.Init(req.ctx, init, func(nctx context.Context, ninit apollows.PayloadInit) error {
		req.ctx, init = nctx, ninit

		return nil
	})
	if err != nil {
		return
	}

	req.writeWebsocketMessage(req.ctx, apollows.OperationConnectionAck, nil)

	return
}

func (req *websocketRequest) readWebsocketStart(msg *apollows.Message) (err error) {
	if !req.init && req.protocol == apollows.WebsocketSubprotocolGraphqlTransportWS {
		return apollows.EventUnauthorized
	}

	req.m.RLock()
	_, ok := req.operations[msg.ID]
	req.m.RUnlock()

	if ok {
		return apollows.NewSubscriberAlreadyExistsError(msg.ID)
	}

	opctx := mutable.NewMutableContext(req.ctx)

	opctx.Set(ContextKeyOperationContext, opctx)
	opctx.Set(ContextKeyOperationID, msg.ID)

	req.m.Lock()
	req.operations[msg.ID] = opctx
	req.m.Unlock()

	req.wg.Add(1)

	go func() {
		var payload apollows.PayloadOperation

		operr := json.Unmarshal(msg.Payload.RawMessage, &payload)
		if operr != nil {
			if req.protocol == apollows.WebsocketSubprotocolGraphqlTransportWS {
				operr = apollows.WrapError(operr, apollows.EventInvalidMessage)
			}
		} else {
			operr = req.server.interceptors.Operation(opctx, &payload, req.serveWebsocketOperation)
		}

		if operr != nil && !ContextOperationExecuted(opctx) {
			req.handleError(opctx, operr)
		}

		if !ContextOperationStopped(opctx) ||
			req.protocol == apollows.WebsocketSubprotocolGraphqlWS {
			req.writeWebsocketMessage(opctx, apollows.OperationComplete, nil)
		}

		opctx.Cancel()

		req.m.Lock()
		delete(req.operations, msg.ID)
		req.m.Unlock()

		req.wg.Done()
	}()

	return
}

func (req *websocketRequest) readWebsocketStop(msg *apollows.Message) (err error) {
	if !req.init && req.protocol == apollows.WebsocketSubprotocolGraphqlTransportWS {
		return apollows.EventUnauthorized
	}

	req.m.RLock()
	prev, ok := req.operations[msg.ID]
	req.m.RUnlock()

	if ok {
		prev.Set(ContextKeyOperationStopped, true)
		prev.Cancel()
	}

	return
}

func (req *websocketRequest) readWebsocketTerminate() (err error) {
	if req.protocol == apollows.WebsocketSubprotocolGraphqlTransportWS {
		return apollows.EventUnauthorized
	}

	RequestContext(req.ctx).Set(ContextKeyOperationStopped, true)

	req.outgoing <- outgoingMessage{
		Error: apollows.EventCloseNormal,
	}

	return
}

func (req *websocketRequest) readWebsocketPing(msg *apollows.Message) {
	req.writeWebsocketMessage(req.ctx, apollows.OperationPong, msg.Payload.Value)
}

func (req *websocketRequest) backgroundTimeout(timeout time.Duration, connectSuccessful chan struct{}) {
	timer := time.NewTimer(timeout)

	select {
	case <-timer.C:
		req.handleError(req.ctx, apollows.EventInitializationTimeout)
	case <-connectSuccessful:
	case <-req.ctx.Done():
	}

	timer.Stop()
}

func (req *websocketRequest) readWebsocket() {
	var err error

	defer func() {
		if err != nil {
			req.handleError(req.ctx, err)
		}

		// cancel request context and consequently all pending operation contexts
		RequestContext(req.ctx).Cancel()

		// await for all operations to complete, so nothing will write to req.outgoing from this point
		req.wg.Wait()

		close(req.outgoing)
	}()

	var connectSuccessful chan struct{}

	if req.server.connectTimeout > 0 {
		connectSuccessful = make(chan struct{})

		go req.backgroundTimeout(req.server.connectTimeout, connectSuccessful)
	}

	for {
		var msg apollows.Message

		err = req.ws.ReadJSON(&msg)
		if err != nil {
			return
		}

		switch msg.Type {
		case apollows.OperationConnectionInit:
			if req.init {
				err = apollows.EventTooManyInitializationRequests

				return
			}

			req.init = true

			if connectSuccessful != nil {
				connectSuccessful <- struct{}{}
				close(connectSuccessful)
			}

			err = req.readWebsocketInit(&msg)
		case apollows.OperationStart, apollows.OperationSubscribe:
			err = req.readWebsocketStart(&msg)
		case apollows.OperationStop, apollows.OperationComplete:
			err = req.readWebsocketStop(&msg)
		case apollows.OperationTerminate:
			err = req.readWebsocketTerminate()
		case apollows.OperationPing:
			req.readWebsocketPing(&msg)
		}

		if err != nil {
			return
		}
	}
}

func (req *websocketRequest) serveWebsocketOperation(
	ctx context.Context,
	payload *apollows.PayloadOperation,
) (err error) {
	err = req.server.interceptors.OperationParse(ctx, payload, req.server.operationParse)
	if err != nil {
		return
	}

	cres, err := req.server.interceptors.OperationExecute(ctx, payload, req.server.operationExecute)
	if err != nil {
		return
	}

	return req.server.processResults(ctx, payload, cres, func(ctx context.Context, result *graphql.Result) error {
		req.writeWebsocketData(ctx, result)

		return nil
	})
}
