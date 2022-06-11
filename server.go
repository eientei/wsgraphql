package wsgraphql

import (
	"errors"
	"net/http"
	"time"

	"github.com/eientei/wsgraphql/mutable"
	"github.com/graphql-go/graphql"
)

var (
	errHTTPQueryRejected = errors.New("HTTP query rejected")
	errReflectExtensions = errors.New("could not reflect schema extensions")
)

type serverConfig struct {
	upgrader          Upgrader
	callbacks         Callbacks
	rejectHTTPQueries bool
	keepalive         time.Duration
}

type serverImpl struct {
	rootObject map[string]interface{}
	serverConfig
	extensions []graphql.Extension
	schema     graphql.Schema
}

func (server *serverImpl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqctx := mutable.NewMutableContext(r.Context())

	reqctx.Set(ContextKeyRequestContext, reqctx)
	reqctx.Set(ContextKeyHTTPRequest, r)
	reqctx.Set(ContextKeyHTTPResponseWriter, w)

	var err error

	defer func() {
		server.callbacks.OnRequestDone(reqctx, r, w, err)

		reqctx.Cancel()
	}()

	err = server.callbacks.OnRequest(reqctx, r, w)
	if err != nil {
		return
	}

	if r.Header.Get("connection") == "" || r.Header.Get("upgrade") == "" || server.upgrader == nil {
		err = server.servePlainRequest(reqctx, w, r)

		return
	}

	err = server.serveWebsocketRequest(reqctx, w, r)
	if err != nil {
		return
	}
}
