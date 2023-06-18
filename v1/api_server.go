package wsgraphql

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"time"
	"unsafe"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

// Server implements graphql http handler with websocket support (if upgrader is provided with WithUpgrader)
type Server interface {
	http.Handler
}

// NewServer returns new Server instance
func NewServer(
	schema graphql.Schema,
	options ...ServerOption,
) (Server, error) {
	var c serverConfig

	c.subscriptionProtocols = make(map[apollows.Protocol]struct{})

	for _, o := range options {
		err := o(&c)
		if err != nil {
			return nil, err
		}
	}

	if len(c.subscriptionProtocols) == 0 {
		c.subscriptionProtocols[apollows.WebsocketSubprotocolGraphqlWS] = struct{}{}
		c.subscriptionProtocols[apollows.WebsocketSubprotocolGraphqlTransportWS] = struct{}{}
	}

	initInterceptors(&c)

	if c.resultProcessor == nil {
		c.resultProcessor = identityResultProcessor
	}

	f := reflect.ValueOf(&schema).Elem().FieldByName("extensions")

	exts, ok := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Elem().Interface().([]graphql.Extension)
	if !ok {
		return nil, errReflectExtensions
	}

	return &serverImpl{
		schema:       schema,
		extensions:   exts,
		serverConfig: c,
	}, nil
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

// WithInterceptors option sets interceptors around various stages of requests
func WithInterceptors(interceptors Interceptors) ServerOption {
	return func(config *serverConfig) error {
		config.interceptors = interceptors

		return nil
	}
}

// WithExtraInterceptors option appends interceptors instead of replacing them
func WithExtraInterceptors(interceptors Interceptors) ServerOption {
	return func(config *serverConfig) error {
		if interceptors.HTTPRequest != nil {
			config.interceptors.HTTPRequest = InterceptorHTTPRequestChain(
				config.interceptors.HTTPRequest,
				interceptors.HTTPRequest,
			)
		}

		if interceptors.Init != nil {
			config.interceptors.Init = InterceptorInitChain(
				config.interceptors.Init,
				interceptors.Init,
			)
		}

		if interceptors.Operation != nil {
			config.interceptors.Operation = InterceptorOperationChain(
				config.interceptors.Operation,
				interceptors.Operation,
			)
		}

		if interceptors.OperationParse != nil {
			config.interceptors.OperationParse = InterceptorOperationParseChain(
				config.interceptors.OperationParse,
				interceptors.OperationParse,
			)
		}

		if interceptors.OperationExecute != nil {
			config.interceptors.OperationExecute = InterceptorOperationExecuteChain(
				config.interceptors.OperationExecute,
				interceptors.OperationExecute,
			)
		}

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

// WithProtocol option sets protocol for this sever to use. May be specified multiple times.
func WithProtocol(protocol apollows.Protocol) ServerOption {
	return func(config *serverConfig) error {
		config.subscriptionProtocols[protocol] = struct{}{}

		return nil
	}
}

// WithConnectTimeout option sets duration within which client is allowed to initialize the connection before being
// disconnected
func WithConnectTimeout(timeout time.Duration) ServerOption {
	return func(config *serverConfig) error {
		config.connectTimeout = timeout

		return nil
	}
}

// WithRootObject provides root object that will be used in root resolvers
func WithRootObject(rootObject map[string]interface{}) ServerOption {
	return func(config *serverConfig) error {
		config.rootObject = rootObject

		return nil
	}
}

// ResultProcessor allows to post-process resolved values
type ResultProcessor func(
	ctx context.Context,
	payload *apollows.PayloadOperation,
	result *graphql.Result,
) *graphql.Result

// WithResultProcessor provides ResultProcessor to post-process resolved values
func WithResultProcessor(proc ResultProcessor) ServerOption {
	return func(config *serverConfig) error {
		config.resultProcessor = proc

		return nil
	}
}

// WriteError helper function writing an error to http.ResponseWriter
func WriteError(ctx context.Context, w http.ResponseWriter, err error) {
	if err == nil || ContextHTTPResponseStarted(ctx) {
		return
	}

	var res ResultError

	if !errors.As(err, &res) {
		err = ResultError{
			Result: &graphql.Result{
				Errors: []gqlerrors.FormattedError{
					gqlerrors.FormatError(err),
				},
			},
		}
	}

	bs := []byte(err.Error())

	w.Header().Set("content-length", strconv.Itoa(len(bs)))
	w.WriteHeader(http.StatusBadRequest)

	_, _ = w.Write(bs)
}

func identityResultProcessor(
	ctx context.Context,
	payload *apollows.PayloadOperation,
	result *graphql.Result,
) *graphql.Result {
	return result
}
