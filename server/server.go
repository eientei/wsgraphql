// graphql websocket http handler implementation
package server

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/eientei/wsgraphql/common"

	"github.com/eientei/wsgraphql/mutcontext"

	"github.com/eientei/wsgraphql/proto"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/graphql-go/graphql/language/kinds"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
)

// the Server itself
type Server struct {
	Upgrader        *websocket.Upgrader
	Schema          *graphql.Schema
	OnConnect       common.FuncConnectCallback
	OnOperation     common.FuncOperationCallback
	OnOperationDone common.FuncOperationDoneCallback
	OnDisconnect    common.FuncDisconnectCallback
	OnPlainInit     common.FuncPlainInit
	OnPlainFail     common.FuncPlainFail
	IgnorePlainHttp bool
	KeepAlive       time.Duration
}

// ticker that also closes on stop
type TickCloser struct {
	Ticker  *time.Ticker
	Stopped chan bool
}

// stop also closes
func (tc *TickCloser) Stop() {
	tc.Ticker.Stop()
	close(tc.Stopped)
}

// internal Event base
type Event interface{}

// posted when Connection negotiation commences
type EventConnectionInit struct {
	Parameters interface{}
}

// posted when Connection termination requested
type EventConnectionTerminate struct {
}

// posted when Connection read was closed
type EventConnectionReadClosed struct {
}

// posted when Connection write was closed
type EventConnectionWriteClosed struct {
}

// posted when Connection timer was closed
type EventConnectionTimerClosed struct {
}

// posted when new operation requested
type EventOperationStart struct {
	Id      interface{}
	Payload *proto.PayloadOperation
}

// posted when operation interruption requested
type EventOperationStop struct {
	Id interface{}
}

// posted when operation completed
type EventOperationComplete struct {
	Id interface{}
}

// base Connection state
type Connection struct {
	Server        *Server
	Subscriptions map[interface{}]*Subscription
	Context       mutcontext.MutableContext
	Outgoing      chan *proto.Message
	Events        chan Event
	TickCloser    *TickCloser
	Stopcounter   int32
}

// base operation/Subscription state
type Subscription struct {
	Id      interface{}
	Payload *proto.PayloadOperation
	Context mutcontext.MutableContext
}

// reading bytes from websocket to messages, posting events
func (conn *Connection) ReadLoop(ws *websocket.Conn) {
	ws.SetReadLimit(-1)
	_ = ws.SetReadDeadline(time.Time{})
	payloadbytes := &proto.PayloadBytes{}
	template := proto.Message{
		Payload: payloadbytes,
	}
	for {
		payloadbytes.Value = nil
		payloadbytes.Bytes = nil
		msg := template

		err := ws.ReadJSON(&msg)
		if err != nil {
			_ = ws.Close()
			conn.Events <- &EventConnectionReadClosed{}
			break
		}
		switch msg.Type {
		case proto.GQLConnectionInit:
			var value interface{}
			if len(payloadbytes.Bytes) > 0 {
				err := json.Unmarshal(payloadbytes.Bytes, &value)
				if err != nil {
					conn.Outgoing <- proto.NewMessage("", err.Error(), proto.GQLConnectionError)
					continue
				}
			}

			conn.Events <- &EventConnectionInit{
				Parameters: value,
			}
		case proto.GQLStart:
			payload := &proto.PayloadOperation{}
			err := json.Unmarshal(payloadbytes.Bytes, payload)
			if err != nil {
				conn.Outgoing <- proto.NewMessage("", err.Error(), proto.GQLError)
				continue
			}
			conn.Events <- &EventOperationStart{
				Id:      msg.Id,
				Payload: payload,
			}
		case proto.GQLStop:
			conn.Events <- &EventOperationStop{
				Id: msg.Id,
			}
		case proto.GQLConnectionTerminate:
			conn.Events <- &EventConnectionTerminate{}
		}
	}
}

// writes messages to websocket
func (conn *Connection) WriteLoop(ws *websocket.Conn) {
	for {
		select {
		case o, ok := <-conn.Outgoing:
			if !ok {
				_ = ws.Close()
				conn.Events <- &EventConnectionWriteClosed{}
			}
			if o.Type == proto.GQLConnectionClose {
				if o.Payload != nil && o.Payload.Value != nil {
					_ = ws.WriteJSON(proto.NewMessage("", o.Payload, proto.GQLConnectionError))
				}
				_ = ws.Close()
				conn.Events <- &EventConnectionWriteClosed{}
				return
			}
			err := ws.WriteJSON(o)
			if err != nil {
				_ = ws.Close()
				conn.Events <- &EventConnectionWriteClosed{}
				return
			}
		case <-conn.Context.Done():
			_ = ws.Close()
			conn.Events <- &EventConnectionWriteClosed{}
			return
		}
	}
}

// helper function to parse request into graphql AST
func MakeAST(query string, schema *graphql.Schema) (*ast.Document, *graphql.Result) {
	src := source.NewSource(&source.Source{
		Body: []byte(query),
		Name: "GraphQL request",
	})
	AST, err := parser.Parse(parser.ParseParams{Source: src})
	if err != nil {
		return AST, &graphql.Result{
			Errors: gqlerrors.FormatErrors(err),
		}
	}
	validationResult := graphql.ValidateDocument(schema, AST, nil)

	if !validationResult.IsValid {
		return AST, &graphql.Result{
			Errors: validationResult.Errors,
		}
	}
	return AST, nil
}

// Event reactor
func (conn *Connection) EventLoop() {
	var err error
loop:
	for raw := range conn.Events {
		switch evt := raw.(type) {
		case *EventConnectionInit:
			if conn.Server.OnConnect != nil {
				err = conn.Server.OnConnect(conn.Context, evt.Parameters)
				if err != nil {
					conn.Outgoing <- proto.NewMessage("", err.Error(), proto.GQLConnectionError)
					continue
				}
			}
			conn.Outgoing <- proto.NewMessage("", nil, proto.GQLConnectionAck)
			conn.Outgoing <- proto.NewMessage("", nil, proto.GQLConnectionKeepAlive)
		case *EventOperationStart:
			ctx := mutcontext.CreateNewCancel(context.WithCancel(conn.Context))
			if conn.Server.OnOperation != nil {
				err = conn.Server.OnOperation(conn.Context, ctx, evt.Payload)
				if err != nil {
					conn.Outgoing <- proto.NewMessage(evt.Id, err.Error(), proto.GQLError)
					continue
				}
			}
			conn.Subscriptions[evt.Id] = &Subscription{
				Id:      evt.Id,
				Payload: evt.Payload,
				Context: ctx,
			}
			atomic.AddInt32(&conn.Stopcounter, -1)
			go func(sub *Subscription) {
				AST, res := MakeAST(sub.Payload.Query, conn.Server.Schema)
				if res != nil {
					conn.Outgoing <- proto.NewMessage(sub.Id, res, proto.GQLData)
					conn.Events <- &EventOperationComplete{
						Id: evt.Id,
					}
					return
				}

				issub := false
				for _, n := range AST.Definitions {
					if n.GetKind() == kinds.OperationDefinition {
						op, ok := n.(*ast.OperationDefinition)
						if !ok {
							continue
						}
						if op.Operation == ast.OperationTypeSubscription {
							issub = true
							break
						}
					}
				}

				for {
					res := graphql.Execute(graphql.ExecuteParams{
						Schema:        *conn.Server.Schema,
						Root:          nil,
						AST:           AST,
						OperationName: sub.Payload.OperationName,
						Args:          sub.Payload.Variables,
						Context:       sub.Context,
					})

					conn.Outgoing <- proto.NewMessage(sub.Id, res, proto.GQLData)

					if len(res.Errors) > 0 || !issub || sub.Context.Err() != nil || sub.Context.Completed() {
						break
					}
				}
				conn.Events <- &EventOperationComplete{
					Id: evt.Id,
				}
			}(conn.Subscriptions[evt.Id])
		case *EventOperationStop:
			if sub, ok := conn.Subscriptions[evt.Id]; ok {
				_ = sub.Context.Cancel()
			}
		case *EventOperationComplete:
			atomic.AddInt32(&conn.Stopcounter, 1)
			if atomic.LoadInt32(&conn.Stopcounter) >= 2 {
				break loop
			}

			conn.Outgoing <- proto.NewMessage(evt.Id, nil, proto.GQLComplete)
			sub := conn.Subscriptions[evt.Id]
			if conn.Server.OnOperationDone != nil {
				err = conn.Server.OnOperationDone(conn.Context, sub.Context, sub.Payload)
				if err != nil {
					conn.Outgoing <- proto.NewMessage("", err.Error(), proto.GQLConnectionClose)
				}
			}
			delete(conn.Subscriptions, evt.Id)
		case *EventConnectionTerminate:
			conn.Outgoing <- proto.NewMessage("", nil, proto.GQLConnectionClose)
		case *EventConnectionWriteClosed:
			atomic.AddInt32(&conn.Stopcounter, 1)
			if conn.TickCloser != nil {
				conn.TickCloser.Stop()
			}
			go func() {
				for range conn.Outgoing {
				}
			}()
			if atomic.LoadInt32(&conn.Stopcounter) >= 2 {
				break loop
			}
		case *EventConnectionReadClosed:
			conn.Context.Cancel()
			atomic.AddInt32(&conn.Stopcounter, 1)
			if atomic.LoadInt32(&conn.Stopcounter) >= 2 {
				break loop
			}
		case *EventConnectionTimerClosed:
			atomic.AddInt32(&conn.Stopcounter, 1)
			if atomic.LoadInt32(&conn.Stopcounter) >= 2 {
				break loop
			}
		}

	}
	close(conn.Events)
	close(conn.Outgoing)
	conn.Context.Complete()
	if conn.Server.OnDisconnect != nil {
		_ = conn.Server.OnDisconnect(conn.Context)
	}
}

// serve plain http request
func (server *Server) ServePlainHTTP(ctx mutcontext.MutableContext, w http.ResponseWriter, r *http.Request) {
	if server.IgnorePlainHttp {
		if server.OnPlainFail != nil {
			server.OnPlainFail(ctx, r, w, common.ErrPlainHttpIgnored)
		}
		return
	}

	var result interface{}
	defer func() {
		if result != nil {
			err := json.NewEncoder(w).Encode(result)
			if err != nil && server.OnPlainFail != nil {
				server.OnPlainFail(ctx, r, w, err)
			}
		}
	}()

	var err error
	if server.OnConnect != nil {
		err = server.OnConnect(ctx, nil)
		if err != nil {
			result = err.Error()
			return
		}
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		if server.OnPlainFail != nil {
			server.OnPlainFail(ctx, r, w, err)
		}
		return
	}

	payload := &proto.PayloadOperation{}
	if err = json.Unmarshal(body, payload); err != nil {
		if server.OnPlainFail != nil {
			server.OnPlainFail(ctx, r, w, err)
		}
		return
	}

	opctx := mutcontext.CreateNewCancel(context.WithCancel(ctx))
	if server.OnOperation != nil {
		err = server.OnOperation(ctx, opctx, payload)
		if err != nil {
			result = err.Error()
			return
		}
	}

	params := graphql.Params{
		Schema:         *server.Schema,
		RequestString:  payload.Query,
		VariableValues: payload.Variables,
		OperationName:  payload.OperationName,
		Context:        ctx,
	}

	result = graphql.Do(params)

	if server.OnOperationDone != nil {
		_ = server.OnOperationDone(ctx, opctx, payload)
	}
	if server.OnDisconnect != nil {
		_ = server.OnDisconnect(ctx)
	}
}

// serve websocket http request
func (server *Server) ServeWebsocketHTTP(ctx mutcontext.MutableContext, w http.ResponseWriter, r *http.Request) {
	ws, err := server.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		if server.OnPlainFail != nil {
			server.OnPlainFail(ctx, r, w, err)
		}
		return
	}
	if ws.Subprotocol() != "graphql-ws" {
		_ = ws.Close()
		return
	}

	var ticker *TickCloser
	if server.KeepAlive > 0 {
		ticker = &TickCloser{
			Ticker:  time.NewTicker(server.KeepAlive),
			Stopped: make(chan bool),
		}
	}

	conn := &Connection{
		Server:        server,
		Subscriptions: make(map[interface{}]*Subscription),
		Context:       ctx,
		Outgoing:      make(chan *proto.Message),
		Events:        make(chan Event),
		TickCloser:    ticker,
	}

	go conn.ReadLoop(ws)
	go conn.WriteLoop(ws)
	go conn.EventLoop()

	if conn.TickCloser != nil {
		atomic.AddInt32(&conn.Stopcounter, -1)
		go func() {
			for {
				select {
				case <-conn.TickCloser.Stopped:
					conn.Events <- &EventConnectionTimerClosed{}
					return
				case <-conn.TickCloser.Ticker.C:
					conn.Outgoing <- proto.NewMessage("", nil, proto.GQLConnectionKeepAlive)
				}
			}
		}()
	}
}

// http.Handler entrypoint
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := mutcontext.CreateNewCancel(context.WithCancel(context.Background()))
	ctx.Set(common.KeyHttpRequest, r)

	if server.OnPlainInit != nil {
		server.OnPlainInit(ctx, r, w)
	}
	h := r.Header

	if h.Get("Connection") == "" || h.Get("Upgrade") == "" || server.Upgrader == nil {
		server.ServePlainHTTP(ctx, w, r)
		return
	}

	server.ServeWebsocketHTTP(ctx, w, r)
}
