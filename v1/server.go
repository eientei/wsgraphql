package wsgraphql

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/graphql-go/graphql/gqlerrors"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/graphql-go/graphql"
)

var (
	errHTTPQueryRejected = errors.New("HTTP query rejected")
	errReflectExtensions = errors.New("could not reflect schema extensions")
)

type serverConfig struct {
	upgrader              Upgrader
	interceptors          Interceptors
	resultProcessor       ResultProcessor
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

func (server *serverImpl) handleHTTPRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
	if r.Header.Get("connection") != "" && r.Header.Get("upgrade") != "" && server.upgrader != nil {
		err = server.serveWebsocketRequest(ctx, w, r)
	} else {
		err = server.servePlainRequest(ctx)
	}

	return
}

func (server *serverImpl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqctx := mutable.NewMutableContext(r.Context())

	reqctx.Set(ContextKeyRequestContext, reqctx)
	reqctx.Set(ContextKeyHTTPRequest, r)
	reqctx.Set(ContextKeyHTTPResponseWriter, w)

	_ = server.interceptors.HTTPRequest(reqctx, w, r, server.handleHTTPRequest)

	reqctx.Cancel()
}

func (server *serverImpl) processResults(
	ctx context.Context,
	payload *apollows.PayloadOperation,
	cres chan *graphql.Result,
	write func(ctx context.Context, result *graphql.Result) error,
) (err error) {
	OperationContext(ctx).Set(ContextKeyOperationExecuted, true)

	for {
		select {
		case <-ctx.Done():
			if !ContextOperationStopped(ctx) {
				err = ctx.Err()
			}

			return
		case result, ok := <-cres:
			if !ok {
				return
			}

			err = server.processResult(ctx, payload, result, write)
			if err != nil {
				return err
			}
		}
	}
}

func (server *serverImpl) processResult(
	ctx context.Context,
	payload *apollows.PayloadOperation,
	result *graphql.Result,
	write func(ctx context.Context, result *graphql.Result) error,
) error {
	result = server.resultProcessor(ctx, payload, result)

	var tgterrs []gqlerrors.FormattedError

	err, ok := result.Data.(error)
	if ok {
		tgterrs = append(tgterrs, FormatError(err))
	}

	for _, src := range result.Errors {
		tgterrs = append(tgterrs, FormatError(src))
	}

	result.Errors = tgterrs

	err = write(ctx, result)
	if err != nil {
		return err
	}

	if result.HasErrors() {
		return ResultError{
			Result: result,
		}
	}

	return nil
}
