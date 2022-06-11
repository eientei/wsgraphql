package wsgraphql

import (
	"context"
	"net/http"
	"reflect"
	"strconv"
	"time"
	"unsafe"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/graphql-go/graphql"
)

// Server implements graphql http handler with websocket support (if upgrader is provided with WithUpgrader)
type Server interface {
	http.Handler
}

// NewServer returns new Server instance
func NewServer(schema graphql.Schema, rootObject map[string]interface{}, options ...ServerOption) (Server, error) {
	var c serverConfig

	for _, o := range options {
		err := o(&c)
		if err != nil {
			return nil, err
		}
	}

	if c.callbacks.OnRequest == nil {
		c.callbacks.OnRequest = func(ctx mutable.Context, r *http.Request, w http.ResponseWriter) error {
			return nil
		}
	}

	if c.callbacks.OnRequestDone == nil {
		c.callbacks.OnRequestDone = func(ctx mutable.Context, r *http.Request, w http.ResponseWriter, err error) {
			WriteError(ctx, w, err)
		}
	}

	if c.callbacks.OnConnect == nil {
		c.callbacks.OnConnect = func(reqctx mutable.Context, init apollows.PayloadInit) error {
			return nil
		}
	}

	if c.callbacks.OnOperation == nil {
		c.callbacks.OnOperation = func(ctx mutable.Context, payload *apollows.PayloadOperation) error {
			return nil
		}
	}

	if c.callbacks.OnOperationResult == nil {
		c.callbacks.OnOperationResult = func(
			ctx mutable.Context,
			payload *apollows.PayloadOperation,
			result *graphql.Result,
		) error {
			return nil
		}
	}

	if c.callbacks.OnOperationDone == nil {
		c.callbacks.OnOperationDone = func(ctx mutable.Context, payload *apollows.PayloadOperation, err error) error {
			return err
		}
	}

	f := reflect.ValueOf(&schema).Elem().FieldByName("extensions")

	exts, ok := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem().Interface().([]graphql.Extension)
	if !ok {
		return nil, errReflectExtensions
	}

	return &serverImpl{
		schema:       schema,
		extensions:   exts,
		rootObject:   rootObject,
		serverConfig: c,
	}, nil
}

// Callbacks supported by the server
// use wsgraphql.ContextHTTPRequest / wsgraphql.ContextHTTPResponseWriter to access underlying
// http.Request and http.ResponseWriter
type Callbacks struct {
	// OnRequest called once HTTP request is received, before attempting to do websocket upgrade or plain request
	// execution, consequently before OnConnect as well.
	OnRequest func(reqctx mutable.Context, r *http.Request, w http.ResponseWriter) error

	// OnRequestDone called once HTTP request is finished, regardless of request type, with error occurred during
	// request execution (if any).
	// By default, if error is present, will write error text and return 400 code.
	OnRequestDone func(reqctx mutable.Context, r *http.Request, w http.ResponseWriter, err error)

	// OnConnect is called once per HTTP request, after websocket upgrade and init message received in case of
	// websocket request, or before execution in case of plain request
	OnConnect func(reqctx mutable.Context, init apollows.PayloadInit) error

	// OnOperation is called before each operation with original payload, allowing to modify it or terminate
	// the operation by returning an error
	OnOperation func(opctx mutable.Context, payload *apollows.PayloadOperation) error

	// OnOperationResult is called after operation result is received, allowing to postprocess it or terminate the
	// operation before returning the result with error
	OnOperationResult func(opctx mutable.Context, payload *apollows.PayloadOperation, result *graphql.Result) error

	// OnOperationDone is called once operation is finished, with error occured during the execution (if any)
	// error returned from this handler will close the websocket / terminate HTTP request with error response.
	// By default, will pass through any error occurred
	OnOperationDone func(opctx mutable.Context, payload *apollows.PayloadOperation, err error) error
}

// ServerOption to configure Server
type ServerOption func(config *serverConfig) error

// WithUpgrader option sets Upgrader (interface in image of gorilla websocket upgrader)
func WithUpgrader(upgrader Upgrader) ServerOption {
	return func(config *serverConfig) error {
		config.upgrader = upgrader

		return nil
	}
}

// WithCallbacks option sets callbacks handling various stages of requests
func WithCallbacks(callbacks Callbacks) ServerOption {
	return func(config *serverConfig) error {
		config.callbacks = callbacks

		return nil
	}
}

// WithKeepalive enabled sending keepalive messages with provided intervals
func WithKeepalive(interval time.Duration) ServerOption {
	return func(config *serverConfig) error {
		config.keepalive = interval

		return nil
	}
}

// WithoutHTTPQueries option prevents HTTP queries from being handled, allowing only websocket queries
func WithoutHTTPQueries() ServerOption {
	return func(config *serverConfig) error {
		config.rejectHTTPQueries = true

		return nil
	}
}

// WriteError helper function writing an error to http.ResponseWriter
func WriteError(ctx context.Context, w http.ResponseWriter, err error) {
	if err == nil || ContextWebsocketConnection(ctx) != nil {
		return
	}

	bs := []byte(err.Error())

	w.Header().Set("content-length", strconv.Itoa(len(bs)))
	w.WriteHeader(http.StatusBadRequest)

	_, _ = w.Write(bs)
}
