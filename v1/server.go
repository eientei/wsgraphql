package wsgraphql

import (
	"errors"
	"net/http"
	"time"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
)

var (
	errHTTPQueryRejected = errors.New("HTTP query rejected")
	errReflectExtensions = errors.New("could not reflect schema extensions")
)

type serverConfig struct {
	upgrader              Upgrader
	callbacks             Callbacks
	rootObject            map[string]interface{}
	subscriptionProtocols map[apollows.Protocol]struct{}
	keepalive             time.Duration
	connectTimeout        time.Duration
	rejectHTTPQueries     bool
}

type serverImpl struct {
	extensions []graphql.Extension
	schema     graphql.Schema
	serverConfig
}

func (server *serverImpl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqctx := mutable.NewMutableContext(r.Context())

	reqctx.Set(ContextKeyRequestContext, reqctx)
	reqctx.Set(ContextKeyHTTPRequest, r)
	reqctx.Set(ContextKeyHTTPResponseWriter, w)

	var err error

	defer func() {
		err = server.callbacks.OnDisconnect(reqctx, err)

		if err != nil {
			if _, ok := err.(resultError); !ok {
				err = resultError{
					Result: &graphql.Result{
						Errors: []gqlerrors.FormattedError{
							gqlerrors.FormatError(err),
						},
					},
				}
			}
		}

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
}
