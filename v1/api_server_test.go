package wsgraphql

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/stretchr/testify/assert"
)

func TestWithCallbacksOnRequest(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())

	var c serverConfig

	var encountered error

	target := errors.New("123")

	assert.NoError(t, WithCallbacks(Callbacks{
		OnRequest: func(reqctx mutable.Context, r *http.Request, w http.ResponseWriter) error {
			return target
		},
		OnRequestDone: func(reqctx mutable.Context, r *http.Request, w http.ResponseWriter, origerr error) {
			encountered = origerr
		},
		OnConnect:             nil,
		OnDisconnect:          nil,
		OnOperation:           nil,
		OnOperationValidation: nil,
		OnOperationResult:     nil,
		OnOperationDone:       nil,
	})(&c))

	w := httptest.NewRecorder()

	r, _ := http.NewRequest(http.MethodGet, "", nil)

	_ = c.interceptors.HTTPRequest(
		opctx,
		w,
		r,
		func(reqctx context.Context, w http.ResponseWriter, r *http.Request) error {
			return nil
		},
	)

	assert.Equal(t, target, encountered)
}

func TestWithCallbacksOnRequestHandler(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())

	var c serverConfig

	var encountered error

	target := errors.New("123")

	assert.NoError(t, WithCallbacks(Callbacks{
		OnRequest: nil,
		OnRequestDone: func(reqctx mutable.Context, r *http.Request, w http.ResponseWriter, origerr error) {
			encountered = origerr
		},
		OnConnect:             nil,
		OnDisconnect:          nil,
		OnOperation:           nil,
		OnOperationValidation: nil,
		OnOperationResult:     nil,
		OnOperationDone:       nil,
	})(&c))

	w := httptest.NewRecorder()

	r, _ := http.NewRequest(http.MethodGet, "", nil)

	_ = c.interceptors.HTTPRequest(
		opctx,
		w,
		r,
		func(reqctx context.Context, w http.ResponseWriter, r *http.Request) error {
			return target
		},
	)

	assert.Equal(t, target, encountered)
}

func TestWithCallbacksOnConnect(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())

	var c serverConfig

	var encountered error

	target := errors.New("123")

	assert.NoError(t, WithCallbacks(Callbacks{
		OnRequest:     nil,
		OnRequestDone: nil,
		OnConnect: func(reqctx mutable.Context, init apollows.PayloadInit) error {
			return target
		},
		OnDisconnect: func(reqctx mutable.Context, origerr error) error {
			encountered = origerr

			return origerr
		},
		OnOperation:           nil,
		OnOperationValidation: nil,
		OnOperationResult:     nil,
		OnOperationDone:       nil,
	})(&c))

	assert.Equal(t, target, c.interceptors.Init(
		opctx,
		nil,
		func(ctx context.Context, init apollows.PayloadInit) error {
			return nil
		},
	))

	assert.Equal(t, target, encountered)
}

func TestWithCallbacksOnConnectHandler(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())

	var c serverConfig

	var encountered error

	target := errors.New("123")

	assert.NoError(t, WithCallbacks(Callbacks{
		OnRequest:     nil,
		OnRequestDone: nil,
		OnConnect:     nil,
		OnDisconnect: func(reqctx mutable.Context, origerr error) error {
			encountered = origerr

			return origerr
		},
		OnOperation:           nil,
		OnOperationValidation: nil,
		OnOperationResult:     nil,
		OnOperationDone:       nil,
	})(&c))

	assert.Equal(t, target, c.interceptors.Init(
		opctx,
		nil,
		func(ctx context.Context, init apollows.PayloadInit) error {
			return target
		},
	))

	assert.Equal(t, target, encountered)
}

func TestWithCallbacksOnOperation(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())

	var c serverConfig

	var encountered error

	target := errors.New("123")

	assert.NoError(t, WithCallbacks(Callbacks{
		OnRequest:     nil,
		OnRequestDone: nil,
		OnConnect:     nil,
		OnDisconnect:  nil,
		OnOperation: func(opctx mutable.Context, payload *apollows.PayloadOperation) error {
			return target
		},
		OnOperationValidation: nil,
		OnOperationResult:     nil,
		OnOperationDone: func(opctx mutable.Context, payload *apollows.PayloadOperation, origerr error) error {
			encountered = origerr

			return origerr
		},
	})(&c))

	assert.Equal(t, target, c.interceptors.Operation(
		opctx,
		nil,
		func(ctx context.Context, payload *apollows.PayloadOperation) error {
			return nil
		},
	))

	assert.Equal(t, target, encountered)
}

func TestWithCallbacksOnOperationHandler(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())

	var c serverConfig

	var encountered error

	target := errors.New("123")

	assert.NoError(t, WithCallbacks(Callbacks{
		OnRequest:             nil,
		OnRequestDone:         nil,
		OnConnect:             nil,
		OnDisconnect:          nil,
		OnOperation:           nil,
		OnOperationValidation: nil,
		OnOperationResult:     nil,
		OnOperationDone: func(opctx mutable.Context, payload *apollows.PayloadOperation, origerr error) error {
			encountered = origerr

			return origerr
		},
	})(&c))

	assert.Equal(t, target, c.interceptors.Operation(
		opctx,
		nil,
		func(ctx context.Context, payload *apollows.PayloadOperation) error {
			return target
		},
	))

	assert.Equal(t, target, encountered)
}

func TestWithCallbacksOnOperationValidation(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())

	var c serverConfig

	target := errors.New("123")

	assert.NoError(t, WithCallbacks(Callbacks{
		OnRequest:     nil,
		OnRequestDone: nil,
		OnConnect:     nil,
		OnDisconnect:  nil,
		OnOperation:   nil,
		OnOperationValidation: func(
			opctx mutable.Context,
			payload *apollows.PayloadOperation,
			result *graphql.Result,
		) error {
			return target
		},
		OnOperationResult: nil,
		OnOperationDone:   nil,
	})(&c))

	assert.Equal(t, target, c.interceptors.OperationParse(
		opctx,
		nil,
		func(ctx context.Context, payload *apollows.PayloadOperation) error {
			return nil
		},
	))
}

func TestWithCallbacksOnOperationExecution(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())

	var c serverConfig

	target := errors.New("123")

	assert.NoError(t, WithCallbacks(Callbacks{
		OnRequest:             nil,
		OnRequestDone:         nil,
		OnConnect:             nil,
		OnDisconnect:          nil,
		OnOperation:           nil,
		OnOperationValidation: nil,
		OnOperationResult: func(
			opctx mutable.Context,
			payload *apollows.PayloadOperation,
			result *graphql.Result,
		) error {
			return target
		},
		OnOperationDone: nil,
	})(&c))

	ch, err := c.interceptors.OperationExecute(
		opctx,
		nil,
		func(ctx context.Context, payload *apollows.PayloadOperation) (chan *graphql.Result, error) {
			tch := make(chan *graphql.Result, 1)

			tch <- &graphql.Result{}
			close(tch)

			return tch, nil
		},
	)

	assert.Nil(t, err)

	res := <-ch

	assert.NotNil(t, target, res)
	assert.Equal(t, target, res.Data)
}

func TestWithCallbacksOnOperationProcess(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())

	var c serverConfig

	target := errors.New("123")

	assert.NoError(t, WithCallbacks(Callbacks{
		OnRequest:             nil,
		OnRequestDone:         nil,
		OnConnect:             nil,
		OnDisconnect:          nil,
		OnOperation:           nil,
		OnOperationValidation: nil,
		OnOperationResult: func(
			opctx mutable.Context,
			payload *apollows.PayloadOperation,
			result *graphql.Result,
		) error {
			result.Data = target

			return nil
		},
		OnOperationDone: nil,
	})(&c))

	ch, err := c.interceptors.OperationExecute(
		opctx,
		nil,
		func(ctx context.Context, payload *apollows.PayloadOperation) (chan *graphql.Result, error) {
			tch := make(chan *graphql.Result, 1)

			tch <- &graphql.Result{}
			close(tch)

			return tch, nil
		},
	)

	assert.Nil(t, err)

	res := <-ch

	assert.NotNil(t, target, res)
	assert.Equal(t, target, res.Data)
}

func TestWithCallbacksOnOperationExecutionHandler(t *testing.T) {
	opctx := mutable.NewMutableContext(context.Background())

	var c serverConfig

	target := errors.New("123")

	assert.NoError(t, WithCallbacks(Callbacks{
		OnRequest:             nil,
		OnRequestDone:         nil,
		OnConnect:             nil,
		OnDisconnect:          nil,
		OnOperation:           nil,
		OnOperationValidation: nil,
		OnOperationResult:     nil,
		OnOperationDone:       nil,
	})(&c))

	ch, err := c.interceptors.OperationExecute(
		opctx,
		nil,
		func(ctx context.Context, payload *apollows.PayloadOperation) (chan *graphql.Result, error) {
			return nil, target
		},
	)

	assert.Nil(t, ch)
	assert.Equal(t, target, err)
}

func TestWithInterceptorChain(t *testing.T) {
	opctx := context.Background()

	var c serverConfig

	type keyT struct{}

	key := keyT{}

	assert.NoError(t, WithInterceptors(Interceptors{
		HTTPRequest: InterceptorHTTPRequestChain(
			func(ctx context.Context, w http.ResponseWriter, r *http.Request, handler HandlerHTTPRequest) error {
				return handler(context.WithValue(ctx, key, 1), w, r)
			},
			func(ctx context.Context, w http.ResponseWriter, r *http.Request, handler HandlerHTTPRequest) error {
				return handler(context.WithValue(ctx, key, ctx.Value(key).(int)+1), w, r)
			},
		),
		Init: InterceptorInitChain(
			func(ctx context.Context, init apollows.PayloadInit, handler HandlerInit) error {
				return handler(context.WithValue(ctx, key, 2), init)
			},
			func(ctx context.Context, init apollows.PayloadInit, handler HandlerInit) error {
				return handler(context.WithValue(ctx, key, ctx.Value(key).(int)+1), init)
			},
		),
		Operation: InterceptorOperationChain(
			func(ctx context.Context, payload *apollows.PayloadOperation, handler HandlerOperation) error {
				return handler(context.WithValue(ctx, key, 3), payload)
			},
			func(ctx context.Context, payload *apollows.PayloadOperation, handler HandlerOperation) error {
				return handler(context.WithValue(ctx, key, ctx.Value(key).(int)+1), payload)
			},
		),
		OperationParse: InterceptorOperationParseChain(
			func(ctx context.Context, payload *apollows.PayloadOperation, handler HandlerOperationParse) error {
				return handler(context.WithValue(ctx, key, 4), payload)
			},
			func(ctx context.Context, payload *apollows.PayloadOperation, handler HandlerOperationParse) error {
				return handler(context.WithValue(ctx, key, ctx.Value(key).(int)+1), payload)
			},
		),
		OperationExecute: InterceptorOperationExecuteChain(
			func(
				ctx context.Context,
				payload *apollows.PayloadOperation,
				handler HandlerOperationExecute,
			) (chan *graphql.Result, error) {
				return handler(context.WithValue(ctx, key, 5), payload)
			},
			func(
				ctx context.Context,
				payload *apollows.PayloadOperation,
				handler HandlerOperationExecute,
			) (chan *graphql.Result, error) {
				return handler(context.WithValue(ctx, key, ctx.Value(key).(int)+1), payload)
			},
		),
	})(&c))

	r, _ := http.NewRequest(http.MethodGet, "", nil)

	var res []int

	_ = c.interceptors.HTTPRequest(
		opctx,
		httptest.NewRecorder(),
		r,
		func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			res = append(res, ctx.Value(key).(int))

			return nil
		},
	)

	_ = c.interceptors.Init(
		opctx,
		nil,
		func(ctx context.Context, init apollows.PayloadInit) error {
			res = append(res, ctx.Value(key).(int))

			return nil
		},
	)

	_ = c.interceptors.Operation(
		opctx,
		nil,
		func(ctx context.Context, payload *apollows.PayloadOperation) error {
			res = append(res, ctx.Value(key).(int))

			return nil
		},
	)

	_ = c.interceptors.OperationParse(
		opctx,
		nil,
		func(ctx context.Context, payload *apollows.PayloadOperation) error {
			res = append(res, ctx.Value(key).(int))

			return nil
		},
	)

	_, _ = c.interceptors.OperationExecute(
		opctx,
		nil,
		func(ctx context.Context, payload *apollows.PayloadOperation) (chan *graphql.Result, error) {
			res = append(res, ctx.Value(key).(int))

			return nil, nil
		},
	)

	assert.Equal(t, []int{2, 3, 4, 5, 6}, res)
}

func TestWithKeepalive(t *testing.T) {
	var c serverConfig

	assert.NoError(t, WithKeepalive(123)(&c))

	assert.Equal(t, time.Duration(123), c.keepalive)
}

func TestWithoutHTTPQueries(t *testing.T) {
	var c serverConfig

	assert.NoError(t, WithoutHTTPQueries()(&c))

	assert.Equal(t, true, c.rejectHTTPQueries)
}

func TestWithRootObject(t *testing.T) {
	var c serverConfig

	obj := make(map[string]interface{})

	assert.NoError(t, WithRootObject(obj)(&c))

	assert.Equal(t, obj, c.rootObject)
}

func TestWriteError(t *testing.T) {
	mutctx := mutable.NewMutableContext(context.Background())

	rec := httptest.NewRecorder()

	WriteError(mutctx, rec, errors.New("123"))

	resp := rec.Result()

	bs, err := io.ReadAll(resp.Body)

	assert.NoError(t, err)
	assert.Equal(t, ResultError{
		Result: &graphql.Result{
			Errors: []gqlerrors.FormattedError{
				gqlerrors.FormatError(errors.New("123")),
			},
		},
	}.Error(), string(bs))

	assert.NoError(t, resp.Body.Close())
}

func TestWriteErrorResponseStarted(t *testing.T) {
	mutctx := mutable.NewMutableContext(context.Background())

	mutctx.Set(ContextKeyHTTPResponseStarted, true)

	rec := httptest.NewRecorder()

	WriteError(mutctx, rec, errors.New("123"))

	resp := rec.Result()

	bs, err := io.ReadAll(resp.Body)

	assert.NoError(t, err)
	assert.Equal(t, "", string(bs))

	assert.NoError(t, resp.Body.Close())
}

func TestOptError(t *testing.T) {
	srv, err := NewServer(testNewSchema(t), func(config *serverConfig) error {
		return errors.New("123")
	})

	assert.Error(t, err)
	assert.Nil(t, srv)
}
