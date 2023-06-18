package wsgraphql

import (
	"context"
	"net/http"

	"github.com/graphql-go/graphql"

	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/graphql-go/graphql/language/ast"
)

type (
	contextKeyRequestContextT      struct{}
	contextKeyOperationContextT    struct{}
	contextKeyOperationStoppedT    struct{}
	contextKeyOperationExecutedT   struct{}
	contextKeyOperationIDT         struct{}
	contextKeyOperationParamsT     struct{}
	contextKeyAstT                 struct{}
	contextKeySubscriptionT        struct{}
	contextKeyHTTPRequestT         struct{}
	contextKeyHTTPResponseWriterT  struct{}
	contextKeyHTTPResponseStartedT struct{}
	contextKeyWebsocketConnectionT struct{}
)

var (
	// ContextKeyRequestContext used to store HTTP request-scoped mutable.Context
	ContextKeyRequestContext = contextKeyRequestContextT{}

	// ContextKeyOperationContext used to store graphql operation-scoped mutable.Context
	ContextKeyOperationContext = contextKeyOperationContextT{}

	// ContextKeyOperationStopped indicates the operation was stopped on client request
	ContextKeyOperationStopped = contextKeyOperationStoppedT{}

	// ContextKeyOperationExecuted indicates the operation was executed
	ContextKeyOperationExecuted = contextKeyOperationExecutedT{}

	// ContextKeyOperationID indicates the operation ID
	ContextKeyOperationID = contextKeyOperationIDT{}

	// ContextKeyOperationParams used to store operation params
	ContextKeyOperationParams = contextKeyOperationParamsT{}

	// ContextKeyAST used to store operation's ast.Document (abstract syntax tree)
	ContextKeyAST = contextKeyAstT{}

	// ContextKeySubscription used to store operation subscription flag
	ContextKeySubscription = contextKeySubscriptionT{}

	// ContextKeyHTTPRequest used to store HTTP request
	ContextKeyHTTPRequest = contextKeyHTTPRequestT{}

	// ContextKeyHTTPResponseWriter used to store HTTP response
	ContextKeyHTTPResponseWriter = contextKeyHTTPResponseWriterT{}

	// ContextKeyHTTPResponseStarted used to indicate HTTP response already has headers sent
	ContextKeyHTTPResponseStarted = contextKeyHTTPResponseStartedT{}

	// ContextKeyWebsocketConnection used to store websocket connection
	ContextKeyWebsocketConnection = contextKeyWebsocketConnectionT{}
)

func defaultMutcontext(ctx context.Context, mutctx mutable.Context) mutable.Context {
	if mutctx != nil {
		return mutctx
	}

	return mutable.NewMutableContext(ctx)
}

// RequestContext returns HTTP request-scoped v1.mutable context from provided context or nil if none present
func RequestContext(ctx context.Context) (mutctx mutable.Context) {
	defer func() {
		mutctx = defaultMutcontext(ctx, mutctx)
	}()

	v := ctx.Value(ContextKeyRequestContext)
	if v == nil {
		return nil
	}

	mutctx, ok := v.(mutable.Context)
	if !ok {
		return nil
	}

	return mutctx
}

// OperationContext returns graphql operation-scoped v1.mutable context from provided context or nil if none present
func OperationContext(ctx context.Context) (mutctx mutable.Context) {
	defer func() {
		mutctx = defaultMutcontext(ctx, mutctx)
	}()

	v := ctx.Value(ContextKeyOperationContext)
	if v == nil {
		return nil
	}

	mutctx, ok := v.(mutable.Context)
	if !ok {
		return nil
	}

	return mutctx
}

// ContextOperationStopped returns true if user requested operation stop
func ContextOperationStopped(ctx context.Context) bool {
	v := ctx.Value(ContextKeyOperationStopped)
	if v == nil {
		return false
	}

	res, ok := v.(bool)
	if !ok {
		return false
	}

	return res
}

// ContextOperationExecuted returns true if user requested operation stop
func ContextOperationExecuted(ctx context.Context) bool {
	v := ctx.Value(ContextKeyOperationExecuted)
	if v == nil {
		return false
	}

	res, ok := v.(bool)
	if !ok {
		return false
	}

	return res
}

// ContextOperationID returns operaion ID stored in the context
func ContextOperationID(ctx context.Context) string {
	v := ctx.Value(ContextKeyOperationID)
	if v == nil {
		return ""
	}

	res, ok := v.(string)
	if !ok {
		return ""
	}

	return res
}

// ContextOperationParams returns operation params stored in the context
func ContextOperationParams(ctx context.Context) (res *graphql.Params) {
	v := ctx.Value(ContextKeyOperationParams)
	if v == nil {
		return &graphql.Params{}
	}

	r, ok := v.(*graphql.Params)
	if !ok || r == nil {
		return &graphql.Params{}
	}

	return r
}

// ContextAST returns operation's abstract syntax tree document
func ContextAST(ctx context.Context) *ast.Document {
	v := ctx.Value(ContextKeyAST)
	if v == nil {
		return nil
	}

	astdoc, ok := v.(*ast.Document)
	if !ok {
		return nil
	}

	return astdoc
}

// ContextSubscription returns operation's subscription flag
func ContextSubscription(ctx context.Context) bool {
	v := ctx.Value(ContextKeySubscription)
	if v == nil {
		return false
	}

	sub, ok := v.(bool)
	if !ok {
		return false
	}

	return sub
}

// ContextHTTPRequest returns http request stored in a context
func ContextHTTPRequest(ctx context.Context) *http.Request {
	v := ctx.Value(ContextKeyHTTPRequest)
	if v == nil {
		return nil
	}

	req, ok := v.(*http.Request)
	if !ok {
		return nil
	}

	return req
}

// ContextHTTPResponseWriter returns http response writer stored in a context
func ContextHTTPResponseWriter(ctx context.Context) http.ResponseWriter {
	v := ctx.Value(ContextKeyHTTPResponseWriter)
	if v == nil {
		return nil
	}

	req, ok := v.(http.ResponseWriter)
	if !ok {
		return nil
	}

	return req
}

// ContextHTTPResponseStarted returns true if HTTP response has already headers sent
func ContextHTTPResponseStarted(ctx context.Context) bool {
	v := ctx.Value(ContextKeyHTTPResponseStarted)
	if v == nil {
		return false
	}

	val, ok := v.(bool)
	if !ok {
		return false
	}

	return val
}

// ContextWebsocketConnection returns websocket connection stored in a context
func ContextWebsocketConnection(ctx context.Context) Conn {
	v := ctx.Value(ContextKeyWebsocketConnection)
	if v == nil {
		return nil
	}

	conn, ok := v.(Conn)
	if !ok {
		return nil
	}

	return conn
}
