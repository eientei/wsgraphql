// graphql over websocket transport using apollo websocket protocol
//
// Usage
//
// When subscription operation is present in query, it is called repeatedly until it would return an error or cancel
// context associated with this operation.
//
// Context provided to operation is also an instance of mutcontext.MutableContext, which supports setting additional
// values on same instance and holds cancel() function within it.
//
// Implementors of subscribable operation are expected to cast provided context to mutcontext.MutableContext and on
// first invocation initialize data persisted across calls to it (like, external connection, database cursor or
// anything like that) as well as cleanup function using mutcontext.MutableContext.SetCleanup(func()), which would be
// called once operation is complete.
//
// After initialization, subscription would be called repeatedly, expected to return next value at each invocation,
// until an error is encountered, interruption is requested, or data is normally exhausted, in which case
// mutcontext.MutableContext is supposed to be issued with .Complete()
//
// After any of that, cleanup function (if any provided) would be called, ensuring operation code could release any
// resources allocated
//
// Non-subscribable operations are straight-forward stateless invocations.
//
// Subscriptions are also not required to be stateful, however that would mean they would return values as fast as
// they could produce them, without any timeouts and delays implemented, which would likely to require some state,
// e.g. time of last invocation.
//
// By default, implementation allows calling any operation once via non-websocket plain http request.
package wsgraphql

import (
	"net/http"
	"reflect"
	"time"

	"github.com/eientei/wsgraphql/server"
	"github.com/graphql-go/graphql"

	"github.com/eientei/wsgraphql/common"

	"github.com/eientei/wsgraphql/mutcontext"

	"github.com/gorilla/websocket"
)

const (
	KeyHttpRequest = common.KeyHttpRequest
)

var (
	ErrSchemaRequired   = common.ErrSchemaRequired
	ErrPlainHttpIgnored = common.ErrPlainHttpIgnored
)

type FuncConnectCallback common.FuncConnectCallback

type FuncOperationCallback common.FuncOperationCallback

type FuncOperationDoneCallback common.FuncOperationDoneCallback

type FuncDisconnectCallback common.FuncDisconnectCallback

type FuncPlainInit common.FuncPlainInit

type FuncPlainFail common.FuncPlainFail

// Config for websocket graphql server
type Config struct {
	// websocket upgrader
	// default: one that simply negotiates 'graphql-ws' protocol
	Upgrader *websocket.Upgrader

	// graphql schema, required
	// default: nil
	Schema *graphql.Schema

	// called when new client is connecting with new parameters or new plain request started
	// default: nothing
	OnConnect FuncConnectCallback

	// called when new operation started
	// default: nothing
	OnOperation FuncOperationCallback

	// called when operation is complete
	// default: nothing
	OnOperationDone FuncOperationDoneCallback

	// called when websocket connection is closed or plain request is served
	// default: nothing
	OnDisconnect FuncDisconnectCallback

	// called when new http connection is established
	// default: nothing
	OnPlainInit FuncPlainInit

	// called when failure occured at plain http stages
	// default: writes back error text
	OnPlainFail FuncPlainFail

	// if true, plain http connections that can't be upgraded would be ignored and not served as one-off requests
	// default: false
	IgnorePlainHttp bool

	// keep alive period, at which server would send keep-alive messages
	// default: 20 seconds
	KeepAlive time.Duration
}

func setDefault(value, def interface{}) {
	v := reflect.ValueOf(value).Elem()
	needdef := false
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		needdef = v.IsNil()
	default:
		zer := reflect.Zero(v.Type())
		needdef = v.Interface() == zer.Interface()
	}

	if needdef {
		v.Set(reflect.ValueOf(def))
	}
}

// Returns new server instance using supplied config (which could be zero-value)
func NewServer(config Config) (http.Handler, error) {
	if config.Schema == nil {
		return nil, ErrSchemaRequired
	}
	setDefault(&config.Upgrader, &websocket.Upgrader{Subprotocols: []string{"graphql-ws"}})
	setDefault(&config.OnPlainFail, func(globalctx mutcontext.MutableContext, r *http.Request, w http.ResponseWriter, err error) {
		_, _ = w.Write([]byte(err.Error()))
	})
	setDefault(&config.KeepAlive, time.Second*20)

	return &server.Server{
		Upgrader:        config.Upgrader,
		Schema:          config.Schema,
		OnConnect:       common.FuncConnectCallback(config.OnConnect),
		OnOperation:     common.FuncOperationCallback(config.OnOperation),
		OnOperationDone: common.FuncOperationDoneCallback(config.OnOperationDone),
		OnDisconnect:    common.FuncDisconnectCallback(config.OnDisconnect),
		OnPlainInit:     common.FuncPlainInit(config.OnPlainInit),
		OnPlainFail:     common.FuncPlainFail(config.OnPlainFail),
		IgnorePlainHttp: config.IgnorePlainHttp,
		KeepAlive:       config.KeepAlive,
	}, nil
}
