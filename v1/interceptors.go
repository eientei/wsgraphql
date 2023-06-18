package wsgraphql

import (
	"context"
	"net/http"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/graphql-go/graphql"
)

// Interceptors allow to customize request processing
// Sequence:
// HTTPRequest -> Init -> [ Operation -> OperationParse -> OperationExecute ]*
type Interceptors struct {
	HTTPRequest      InterceptorHTTPRequest
	Init             InterceptorInit
	Operation        InterceptorOperation
	OperationParse   InterceptorOperationParse
	OperationExecute InterceptorOperationExecute
}

type (
	// HandlerHTTPRequest handler
	HandlerHTTPRequest func(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	// HandlerInit handler
	HandlerInit func(ctx context.Context, init apollows.PayloadInit) error
	// HandlerOperation handler
	HandlerOperation func(ctx context.Context, payload *apollows.PayloadOperation) error
	// HandlerOperationParse handler
	HandlerOperationParse func(ctx context.Context, payload *apollows.PayloadOperation) error
	// HandlerOperationExecute handler
	HandlerOperationExecute func(ctx context.Context, payload *apollows.PayloadOperation) (chan *graphql.Result, error)
)

type (
	// InterceptorHTTPRequest interceptor
	InterceptorHTTPRequest func(
		ctx context.Context,
		w http.ResponseWriter,
		r *http.Request,
		handler HandlerHTTPRequest,
	) error
	// InterceptorInit interceptor
	InterceptorInit func(
		ctx context.Context,
		init apollows.PayloadInit,
		handler HandlerInit,
	) error
	// InterceptorOperation interceptor
	InterceptorOperation func(
		ctx context.Context,
		payload *apollows.PayloadOperation,
		handler HandlerOperation,
	) error
	// InterceptorOperationParse interceptor
	InterceptorOperationParse func(
		ctx context.Context,
		payload *apollows.PayloadOperation,
		handler HandlerOperationParse,
	) error
	// InterceptorOperationExecute interceptor
	InterceptorOperationExecute func(
		ctx context.Context,
		payload *apollows.PayloadOperation,
		handler HandlerOperationExecute,
	) (chan *graphql.Result, error)
)

func interceptorHTTPRequestChain(
	interceptors []InterceptorHTTPRequest,
	idx int,
	handler HandlerHTTPRequest,
) HandlerHTTPRequest {
	for idx < len(interceptors) && interceptors[idx] == nil {
		idx++
	}

	if idx == len(interceptors) {
		return handler
	}

	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return interceptors[idx](ctx, w, r, interceptorHTTPRequestChain(interceptors, idx+1, handler))
	}
}

// InterceptorHTTPRequestChain returns interceptor composed of the provided list
func InterceptorHTTPRequestChain(interceptors ...InterceptorHTTPRequest) InterceptorHTTPRequest {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, handler HandlerHTTPRequest) error {
		return interceptorHTTPRequestChain(interceptors, 0, handler)(ctx, w, r)
	}
}

func interceptorInitChain(
	interceptors []InterceptorInit,
	idx int,
	handler HandlerInit,
) HandlerInit {
	for idx < len(interceptors) && interceptors[idx] == nil {
		idx++
	}

	if idx == len(interceptors) {
		return handler
	}

	return func(ctx context.Context, init apollows.PayloadInit) error {
		return interceptors[idx](ctx, init, interceptorInitChain(interceptors, idx+1, handler))
	}
}

// InterceptorInitChain returns interceptor composed of the provided list
func InterceptorInitChain(interceptors ...InterceptorInit) InterceptorInit {
	return func(ctx context.Context, init apollows.PayloadInit, handler HandlerInit) error {
		return interceptorInitChain(interceptors, 0, handler)(ctx, init)
	}
}

func interceptorOperationChain(
	interceptors []InterceptorOperation,
	idx int,
	handler HandlerOperation,
) HandlerOperation {
	for idx < len(interceptors) && interceptors[idx] == nil {
		idx++
	}

	if idx == len(interceptors) {
		return handler
	}

	return func(ctx context.Context, payload *apollows.PayloadOperation) error {
		return interceptors[idx](ctx, payload, interceptorOperationChain(interceptors, idx+1, handler))
	}
}

// InterceptorOperationChain returns interceptor composed of the provided list
func InterceptorOperationChain(interceptors ...InterceptorOperation) InterceptorOperation {
	return func(ctx context.Context, payload *apollows.PayloadOperation, handler HandlerOperation) error {
		return interceptorOperationChain(interceptors, 0, handler)(ctx, payload)
	}
}

func interceptorOperationParseChain(
	interceptors []InterceptorOperationParse,
	idx int,
	handler HandlerOperationParse,
) HandlerOperationParse {
	for idx < len(interceptors) && interceptors[idx] == nil {
		idx++
	}

	if idx == len(interceptors) {
		return handler
	}

	return func(ctx context.Context, payload *apollows.PayloadOperation) error {
		return interceptors[idx](ctx, payload, interceptorOperationParseChain(interceptors, idx+1, handler))
	}
}

// InterceptorOperationParseChain returns interceptor composed of the provided list
func InterceptorOperationParseChain(interceptors ...InterceptorOperationParse) InterceptorOperationParse {
	return func(ctx context.Context, payload *apollows.PayloadOperation, handler HandlerOperationParse) error {
		return interceptorOperationParseChain(interceptors, 0, handler)(ctx, payload)
	}
}

func interceptorOperationExecuteChain(
	interceptors []InterceptorOperationExecute,
	idx int,
	handler HandlerOperationExecute,
) HandlerOperationExecute {
	for idx < len(interceptors) && interceptors[idx] == nil {
		idx++
	}

	if idx == len(interceptors) {
		return handler
	}

	return func(ctx context.Context, payload *apollows.PayloadOperation) (chan *graphql.Result, error) {
		return interceptors[idx](ctx, payload, interceptorOperationExecuteChain(interceptors, idx+1, handler))
	}
}

// InterceptorOperationExecuteChain returns interceptor composed of the provided list
func InterceptorOperationExecuteChain(interceptors ...InterceptorOperationExecute) InterceptorOperationExecute {
	return func(
		ctx context.Context,
		payload *apollows.PayloadOperation,
		handler HandlerOperationExecute,
	) (chan *graphql.Result, error) {
		return interceptorOperationExecuteChain(interceptors, 0, handler)(ctx, payload)
	}
}

func initInterceptors(c *serverConfig) {
	if c.interceptors.HTTPRequest == nil {
		c.interceptors.HTTPRequest = func(
			ctx context.Context,
			w http.ResponseWriter,
			r *http.Request,
			handler HandlerHTTPRequest,
		) error {
			err := handler(ctx, w, r)
			if err != nil {
				WriteError(ctx, w, err)
			}

			return err
		}
	}

	if c.interceptors.Init == nil {
		c.interceptors.Init = func(
			ctx context.Context,
			init apollows.PayloadInit,
			handler HandlerInit,
		) error {
			return handler(ctx, init)
		}
	}

	if c.interceptors.Operation == nil {
		c.interceptors.Operation = func(
			ctx context.Context,
			payload *apollows.PayloadOperation,
			handler HandlerOperation,
		) error {
			return handler(ctx, payload)
		}
	}

	if c.interceptors.OperationParse == nil {
		c.interceptors.OperationParse = func(
			ctx context.Context,
			payload *apollows.PayloadOperation,
			handler HandlerOperationParse,
		) error {
			return handler(ctx, payload)
		}
	}

	if c.interceptors.OperationExecute == nil {
		c.interceptors.OperationExecute = func(
			ctx context.Context,
			payload *apollows.PayloadOperation,
			handler HandlerOperationExecute,
		) (chan *graphql.Result, error) {
			return handler(ctx, payload)
		}
	}
}
