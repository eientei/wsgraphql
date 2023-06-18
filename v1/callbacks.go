package wsgraphql

import (
	"context"
	"net/http"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/graphql-go/graphql"
)

// WithCallbacks option sets callbacks handling various stages of requests
// Deprecated: use WithInterceptors / WithResultProcessor
func WithCallbacks(callbacks Callbacks) ServerOption {
	return WithExtraInterceptors(Interceptors{
		HTTPRequest:      legacyCallbackHTTPRequest(callbacks),
		Init:             legacyCallbackInit(callbacks),
		Operation:        legacyCallbackOperation(callbacks),
		OperationParse:   legacyCallbackOperationParse(callbacks),
		OperationExecute: legacyCallbackExecute(callbacks),
	})
}

// Callbacks supported by the server
// use wsgraphql.ContextHTTPRequest / wsgraphql.ContextHTTPResponseWriter to access underlying
// http.Request and http.ResponseWriter
// Sequence:
// OnRequest -> OnConnect ->
// [ OnOperation -> OnOperationValidation -> OnOperationResult -> OnOperationDone ]* ->
// OnDisconnect -> OnRequestDone
// Deprecated: use Interceptors / ResultProcessor
type Callbacks struct {
	// OnRequest called once HTTP request is received, before attempting to do websocket upgrade or plain request
	// execution, consequently before OnConnect as well.
	OnRequest func(reqctx mutable.Context, r *http.Request, w http.ResponseWriter) error

	// OnRequestDone called once HTTP request is finished, regardless of request type, with error occurred during
	// request execution (if any).
	// By default, if error is present, will write error text and return 400 code.
	OnRequestDone func(reqctx mutable.Context, r *http.Request, w http.ResponseWriter, origerr error)

	// OnConnect is called once per HTTP request, after websocket upgrade and init message received in case of
	// websocket request, or before execution in case of plain request
	OnConnect func(reqctx mutable.Context, init apollows.PayloadInit) error

	// OnDisconnect is called once per HTTP request, before OnRequestDone, without responsibility to handle errors
	OnDisconnect func(reqctx mutable.Context, origerr error) error

	// OnOperation is called before each operation with original payload, allowing to modify it or terminate
	// the operation by returning an error.
	OnOperation func(opctx mutable.Context, payload *apollows.PayloadOperation) error

	// OnOperationValidation is called after parsing an operation payload with any immediate validation result, if
	// available. AST will be available in context with ContextAST if parsing succeeded.
	OnOperationValidation func(opctx mutable.Context, payload *apollows.PayloadOperation, result *graphql.Result) error

	// OnOperationResult is called after operation result is received, allowing to postprocess it or terminate the
	// operation before returning the result with error. AST is available in context with ContextAST.
	OnOperationResult func(opctx mutable.Context, payload *apollows.PayloadOperation, result *graphql.Result) error

	// OnOperationDone is called once operation is finished, with error occurred during the execution (if any)
	// error returned from this handler will close the websocket / terminate HTTP request with error response.
	// By default, will pass through any error occurred. AST will be available in context with ContextAST if can be
	// parsed.
	OnOperationDone func(opctx mutable.Context, payload *apollows.PayloadOperation, origerr error) error
}

func legacyCallbackExecute(callbacks Callbacks) InterceptorOperationExecute {
	return func(
		ctx context.Context,
		payload *apollows.PayloadOperation,
		handler HandlerOperationExecute,
	) (chan *graphql.Result, error) {
		cres, err := handler(ctx, payload)
		if err != nil {
			return nil, err
		}

		if callbacks.OnOperationResult == nil {
			return cres, nil
		}

		ch := make(chan *graphql.Result)

		go func() {
			defer close(ch)

			for res := range cres {
				err = callbacks.OnOperationResult(OperationContext(ctx), payload, res)
				if err != nil {
					ch <- &graphql.Result{
						Data: err,
					}

					return
				}

				ch <- res
			}
		}()

		return ch, nil
	}
}

func legacyCallbackOperationParse(callbacks Callbacks) InterceptorOperationParse {
	return func(
		ctx context.Context,
		payload *apollows.PayloadOperation,
		handler HandlerOperationParse,
	) error {
		err := handler(ctx, payload)

		if callbacks.OnOperationValidation != nil {
			result, _ := err.(ResultError)

			nerr := callbacks.OnOperationValidation(OperationContext(ctx), payload, result.Result)
			if nerr != nil {
				return nerr
			}
		}

		return err
	}
}

func legacyCallbackOperation(callbacks Callbacks) InterceptorOperation {
	return func(
		ctx context.Context,
		payload *apollows.PayloadOperation,
		handler HandlerOperation,
	) (err error) {
		if callbacks.OnOperation != nil {
			err = callbacks.OnOperation(OperationContext(ctx), payload)
		}

		if err == nil {
			err = handler(ctx, payload)
		}

		if callbacks.OnOperationDone != nil {
			err = callbacks.OnOperationDone(OperationContext(ctx), payload, err)
		}

		return err
	}
}

func legacyCallbackInit(callbacks Callbacks) InterceptorInit {
	return func(
		ctx context.Context,
		init apollows.PayloadInit,
		handler HandlerInit,
	) (err error) {
		if callbacks.OnConnect != nil {
			err = callbacks.OnConnect(RequestContext(ctx), init)
		}

		if err == nil {
			err = handler(ctx, init)
		}

		if callbacks.OnDisconnect != nil {
			err = callbacks.OnDisconnect(RequestContext(ctx), err)
		}

		return err
	}
}

func legacyCallbackHTTPRequest(callbacks Callbacks) InterceptorHTTPRequest {
	return func(
		ctx context.Context,
		w http.ResponseWriter,
		r *http.Request,
		handler HandlerHTTPRequest,
	) error {
		var err error

		defer func() {
			if err != nil {
				WriteError(ctx, w, err)
			}
		}()

		if callbacks.OnRequest != nil {
			err = callbacks.OnRequest(RequestContext(ctx), r, w)
		}

		if err == nil {
			err = handler(ctx, w, r)
		}

		if callbacks.OnRequestDone != nil {
			callbacks.OnRequestDone(RequestContext(ctx), r, w, err)
		} else {
			WriteError(ctx, w, err)
		}

		return err
	}
}
