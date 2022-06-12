package wsgraphql

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/stretchr/testify/assert"
)

func TestWithCallbacks(t *testing.T) {
	var c serverConfig

	cb := Callbacks{
		OnRequest:     nil,
		OnRequestDone: nil,
		OnConnect: func(reqctx mutable.Context, init apollows.PayloadInit) error {
			return errors.New("123")
		},
		OnOperation:       nil,
		OnOperationResult: nil,
		OnOperationDone:   nil,
	}

	assert.NoError(t, WithCallbacks(cb)(&c))

	assert.Equal(t, errors.New("123"), c.callbacks.OnConnect(nil, nil))
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

	bs, err := ioutil.ReadAll(resp.Body)

	assert.NoError(t, err)
	assert.Equal(t, "123", string(bs))

	assert.NoError(t, resp.Body.Close())
}

func TestWriteErrorResponseStarted(t *testing.T) {
	mutctx := mutable.NewMutableContext(context.Background())

	mutctx.Set(ContextKeyHTTPResponseStarted, true)

	rec := httptest.NewRecorder()

	WriteError(mutctx, rec, errors.New("123"))

	resp := rec.Result()

	bs, err := ioutil.ReadAll(resp.Body)

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
