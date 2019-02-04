package common

import (
	"errors"
	"net/http"

	"github.com/eientei/wsgraphql/mutcontext"
	"github.com/eientei/wsgraphql/proto"
)

const (
	// context key for http request
	KeyHttpRequest = "wsgraphql_http_request"
)

var (
	// schema requirement error
	ErrSchemaRequired = errors.New("schema is required")

	// error issued when plain http request is ignored, for FuncPlainFail callback
	ErrPlainHttpIgnored = errors.New("plain http request ignored")
)

// prototype for OnConnect callback
type FuncConnectCallback func(globalctx mutcontext.MutableContext, parameters interface{}) error

// prototype for OnOperation callback
type FuncOperationCallback func(globalctx, opctx mutcontext.MutableContext, operation *proto.PayloadOperation) error

// prototype for OnOperationDone callback
type FuncOperationDoneCallback func(globalctx, opctx mutcontext.MutableContext, operation *proto.PayloadOperation) error

// prototype for OnDisconnect callback
type FuncDisconnectCallback func(globalctx mutcontext.MutableContext) error

// prototype for OnPlainFail callback
type FuncPlainFail func(globalctx mutcontext.MutableContext, r *http.Request, w http.ResponseWriter, err error)

// prototype for OnPlainInit callback
type FuncPlainInit func(globalctx mutcontext.MutableContext, r *http.Request, w http.ResponseWriter)
