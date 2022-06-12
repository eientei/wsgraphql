package wsgraphql

import (
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
	ctx        mutable.Context
	outgoing   chan outgoingMessage
	operations map[string]mutable.Context
	ws         Conn
	server     *serverImpl
	wg         sync.WaitGroup
	m          sync.RWMutex
	init       bool
}

type outgoingMessage struct {
	*apollows.Message
	apollows.Error
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

	req := &websocketRequest{
		ctx:        reqctx,
		outgoing:   make(chan outgoingMessage),
		operations: make(map[string]mutable.Context),
		ws:         ws,
		server:     server,
	}

	reqctx.Set(ContextKeyWebsocketConnection, ws)
	reqctx.Set(ContextKeyHTTPResponseStarted, true)

	var tickerType apollows.Operation

	switch server.subscriptionProtocol {
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

	for {
		select {
		case msg, ok := <-req.outgoing:
			if !ok {
				return
			}

			switch {
			case msg.Message != nil:
				err = ws.WriteJSON(msg.Message)
				if err != nil {
					_ = req.ws.Close(int(apollows.EventCloseError), err.Error())
				}
			case msg.Error != nil:
				err = ws.Close(int(msg.Error.EventMessageType()), msg.Error.Error())
				if err != nil {
					_ = req.ws.Close(int(apollows.EventCloseError), err.Error())
				}
			}
		case <-tickerch:
			err = ws.WriteJSON(&apollows.Message{
				Type: tickerType,
			})
			if err != nil {
				_ = req.ws.Close(int(apollows.EventCloseError), err.Error())
			}
		}
	}
}

func combineErrors(errs []gqlerrors.FormattedError) gqlerrors.FormattedError {
	if len(errs) == 1 {
		return errs[0]
	}

	errmsg := "preparing operation"

	var errmsgs []string

	for _, err := range errs {
		errmsgs = append(errmsgs, err.Error())
	}

	if len(errmsgs) > 0 {
		errmsg += ": " + strings.Join(errmsgs, "; ")
	}

	rooterr := gqlerrors.NewFormattedError(errmsg)

	if len(errs) > 0 {
		rooterr.Extensions["errors"] = errs
	}

	return rooterr
}

func (req *websocketRequest) handleError(ctx mutable.Context, err error, execution bool) {
	awerr, ok := err.(apollows.Error)
	if ok {
		if req.server.subscriptionProtocol == apollows.WebsocketSubprotocolGraphqlWS {
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

	res, ok := err.(resultError)

	if ok {
		switch {
		case execution:
			req.writeWebsocketData(ctx, res.Result)
		case req.server.subscriptionProtocol == apollows.WebsocketSubprotocolGraphqlWS:
			req.writeWebsocketMessage(ctx, apollows.OperationError, combineErrors(res.Result.Errors))
		default:
			req.writeWebsocketMessage(ctx, apollows.OperationError, res.Result.Errors)
		}

		return
	}

	req.writeWebsocketMessage(ctx, apollows.OperationError, gqlerrors.FormatError(err))
}

func (req *websocketRequest) writeWebsocketData(ctx mutable.Context, data *graphql.Result) {
	var t apollows.Operation

	switch req.server.subscriptionProtocol {
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

func (req *websocketRequest) writeWebsocketMessage(ctx mutable.Context, t apollows.Operation, data interface{}) {
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

	err = json.Unmarshal(msg.Payload.RawMessage, &init)
	if err != nil {
		return
	}

	err = req.server.callbacks.OnConnect(req.ctx, init)
	if err != nil {
		return
	}

	req.writeWebsocketMessage(req.ctx, apollows.OperationConnectionAck, nil)

	return
}

func (req *websocketRequest) readWebsocketStart(msg *apollows.Message) (err error) {
	if !req.init && req.server.subscriptionProtocol == apollows.WebsocketSubprotocolGraphqlTransportWS {
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
		executed, operr := req.serveWebsocketOperation(opctx, msg)

		if operr != nil {
			req.handleError(opctx, operr, executed)
		}

		if !ContextOperationStopped(opctx) ||
			req.server.subscriptionProtocol == apollows.WebsocketSubprotocolGraphqlWS {
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
	if !req.init && req.server.subscriptionProtocol == apollows.WebsocketSubprotocolGraphqlTransportWS {
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
	if req.server.subscriptionProtocol == apollows.WebsocketSubprotocolGraphqlTransportWS {
		return apollows.EventUnauthorized
	}

	req.ctx.Set(ContextKeyOperationStopped, true)

	req.outgoing <- outgoingMessage{
		Error: apollows.EventCloseNormal,
	}

	return
}

func (req *websocketRequest) readWebsocket() {
	var err error

	defer func() {
		if err != nil {
			req.handleError(req.ctx, err, false)
		}

		// cancel request context and consequently all pending operation contexts
		req.ctx.Cancel()

		// await for all operations to complete, so nothing will write to req.outgoing from this point
		req.wg.Wait()

		close(req.outgoing)
	}()

	var timer *time.Timer

	if req.server.connectTimeout > 0 {
		timer = time.NewTimer(req.server.connectTimeout)

		go func() {
			select {
			case <-timer.C:
				req.handleError(req.ctx, apollows.EventInitializationTimeout, false)
			case <-req.ctx.Done():
			}
		}()
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

			err = req.readWebsocketInit(&msg)

			if timer != nil {
				timer.Stop()
			}
		case apollows.OperationStart, apollows.OperationSubscribe:
			err = req.readWebsocketStart(&msg)
		case apollows.OperationStop, apollows.OperationComplete:
			err = req.readWebsocketStop(&msg)
		case apollows.OperationTerminate:
			err = req.readWebsocketTerminate()
		}

		if err != nil {
			return
		}
	}
}

func (req *websocketRequest) serveWebsocketOperation(
	opctx mutable.Context,
	msg *apollows.Message,
) (executed bool, err error) {
	var payload apollows.PayloadOperation

	err = json.Unmarshal(msg.Payload.RawMessage, &payload)
	if err != nil {
		if req.server.subscriptionProtocol == apollows.WebsocketSubprotocolGraphqlTransportWS {
			err = apollows.WrapError(err, apollows.EventInvalidMessage)
		}

		return
	}

	defer func() {
		err = req.server.callbacks.OnOperationDone(opctx, &payload, err)
	}()

	err = req.server.callbacks.OnOperation(opctx, &payload)
	if err != nil {
		return
	}

	params, astdoc, subscription, result := req.server.parseAST(opctx, &payload)

	err = req.server.callbacks.OnOperationValidation(opctx, &payload, result)
	if err != nil {
		return
	}

	if result != nil {
		err = resultError{
			Result: result,
		}

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

	executed = true

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

			req.writeWebsocketData(opctx, result)
		}

		if result.HasErrors() {
			return
		}
	}
}
