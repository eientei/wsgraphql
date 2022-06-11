package wsgraphql

import (
	"context"
	"net/http"
	"testing"

	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/stretchr/testify/assert"
)

func TestRequestContext(t *testing.T) {
	reqctx := mutable.NewMutableContext(context.Background())

	mutctx := mutable.NewMutableContext(context.Background())

	r := RequestContext(mutctx)

	assert.NotNil(t, r)
	assert.NotEqual(t, reqctx, r)

	mutctx.Set(ContextKeyRequestContext, reqctx)

	assert.Equal(t, reqctx, RequestContext(mutctx))

	mutctx.Set(ContextKeyRequestContext, 123)

	r = RequestContext(mutctx)

	assert.NotNil(t, r)
	assert.NotEqual(t, reqctx, r)
}

func TestOperationContext(t *testing.T) {
	reqctx := mutable.NewMutableContext(context.Background())

	mutctx := mutable.NewMutableContext(context.Background())

	r := OperationContext(mutctx)

	assert.NotNil(t, r)
	assert.NotEqual(t, reqctx, r)

	mutctx.Set(ContextKeyOperationContext, reqctx)

	assert.Equal(t, reqctx, OperationContext(mutctx))

	mutctx.Set(ContextKeyOperationContext, 123)

	r = OperationContext(mutctx)

	assert.NotNil(t, r)
	assert.NotEqual(t, reqctx, r)
}

func TestContextHTTPResponseWriter(t *testing.T) {
	var h struct {
		http.ResponseWriter
	}

	mutctx := mutable.NewMutableContext(context.Background())

	assert.Nil(t, ContextHTTPResponseWriter(mutctx))

	mutctx.Set(ContextKeyHTTPResponseWriter, h)

	assert.Equal(t, h, ContextHTTPResponseWriter(mutctx))

	mutctx.Set(ContextKeyHTTPResponseWriter, 123)

	assert.Nil(t, ContextHTTPResponseWriter(mutctx))
}

func TestContextHTTPRequest(t *testing.T) {
	h := &http.Request{}

	mutctx := mutable.NewMutableContext(context.Background())

	assert.Nil(t, ContextHTTPRequest(mutctx))

	mutctx.Set(ContextKeyHTTPRequest, h)

	assert.Equal(t, h, ContextHTTPRequest(mutctx))

	mutctx.Set(ContextKeyHTTPRequest, 123)

	assert.Nil(t, ContextHTTPRequest(mutctx))
}

func TestContextSubscription(t *testing.T) {
	mutctx := mutable.NewMutableContext(context.Background())

	assert.Equal(t, false, ContextSubscription(mutctx))

	mutctx.Set(ContextKeySubscription, true)

	assert.Equal(t, true, ContextSubscription(mutctx))

	mutctx.Set(ContextKeySubscription, 123)

	assert.Equal(t, false, ContextSubscription(mutctx))
}

func TestContextAST(t *testing.T) {
	doc := &ast.Document{}

	mutctx := mutable.NewMutableContext(context.Background())

	assert.Nil(t, ContextAST(mutctx))

	mutctx.Set(ContextKeyAST, doc)

	assert.Equal(t, doc, ContextAST(mutctx))

	mutctx.Set(ContextKeyAST, 123)

	assert.Nil(t, ContextAST(mutctx))
}

func TestContextWebsocketConnection(t *testing.T) {
	var conn struct {
		Conn
	}

	mutctx := mutable.NewMutableContext(context.Background())

	assert.Nil(t, ContextWebsocketConnection(mutctx))

	mutctx.Set(ContextKeyWebsocketConnection, conn)

	assert.Equal(t, conn, ContextWebsocketConnection(mutctx))

	mutctx.Set(ContextKeyWebsocketConnection, 123)

	assert.Nil(t, ContextWebsocketConnection(mutctx))
}

func TestContextOperationStopped(t *testing.T) {
	mutctx := mutable.NewMutableContext(context.Background())

	assert.Equal(t, false, ContextOperationStopped(mutctx))

	mutctx.Set(ContextKeyOperationStopped, true)

	assert.Equal(t, true, ContextOperationStopped(mutctx))

	mutctx.Set(ContextKeyOperationStopped, 123)

	assert.Equal(t, false, ContextOperationStopped(mutctx))
}
