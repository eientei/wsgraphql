package wsgraphql

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/graphql-go/graphql"
)

// ResultError passes error result as error
type ResultError struct {
	*graphql.Result
}

// Error implementation
func (r ResultError) Error() string {
	bs, _ := json.Marshal(r.Result)

	return string(bs)
}

func (server *serverImpl) operationExecute(
	ctx context.Context,
	payload *apollows.PayloadOperation,
) (cres chan *graphql.Result, err error) {
	subscription := ContextSubscription(ctx)
	astdoc := ContextAST(ctx)

	if subscription {
		cres = graphql.ExecuteSubscription(graphql.ExecuteParams{
			Schema:        server.schema,
			Root:          server.rootObject,
			AST:           astdoc,
			OperationName: payload.OperationName,
			Args:          payload.Variables,
			Context:       ctx,
		})
	} else {
		cres = make(chan *graphql.Result, 1)
		cres <- graphql.Execute(graphql.ExecuteParams{
			Schema:        server.schema,
			Root:          server.rootObject,
			AST:           astdoc,
			OperationName: payload.OperationName,
			Args:          payload.Variables,
			Context:       ctx,
		})
		close(cres)
	}

	return
}

func (server *serverImpl) operationParse(
	ctx context.Context,
	payload *apollows.PayloadOperation,
) (err error) {
	err = server.parseAST(ctx, payload)
	if err != nil {
		return err
	}

	return
}

func (server *serverImpl) plainRequestOperation(
	ctx context.Context,
	payload *apollows.PayloadOperation,
) (err error) {
	err = server.interceptors.OperationParse(ctx, payload, server.operationParse)
	if err != nil {
		return err
	}

	w := ContextHTTPResponseWriter(ctx)

	w.Header().Set("content-type", "application/json")

	cres, err := server.interceptors.OperationExecute(
		ctx,
		payload,
		server.operationExecute,
	)
	if err != nil {
		return err
	}

	var flusher http.Flusher

	if ContextSubscription(ctx) {
		flusher, _ = w.(http.Flusher)
		w.Header().Set("x-content-type-options", "nosniff")
		w.Header().Set("connection", "keep-alive")
	}

	return server.processResults(ctx, payload, cres, func(ctx context.Context, result *graphql.Result) error {
		return server.writePlainResult(ctx, result, w, flusher)
	})
}

func (server *serverImpl) servePlainRequest(reqctx context.Context) (err error) {
	if server.rejectHTTPQueries {
		return errHTTPQueryRejected
	}

	err = server.interceptors.Init(reqctx, nil, func(nctx context.Context, init apollows.PayloadInit) error {
		reqctx = nctx

		return nil
	})
	if err != nil {
		return err
	}

	var payload apollows.PayloadOperation

	err = json.NewDecoder(ContextHTTPRequest(reqctx).Body).Decode(&payload)
	if err != nil {
		return
	}

	opctx := mutable.NewMutableContext(reqctx)
	opctx.Set(ContextKeyOperationContext, opctx)

	defer opctx.Cancel()

	return server.interceptors.Operation(opctx, &payload, server.plainRequestOperation)
}

func (server *serverImpl) writePlainResult(
	reqctx context.Context,
	result *graphql.Result,
	w http.ResponseWriter,
	flusher http.Flusher,
) (err error) {
	if result == nil {
		return nil
	}

	bs, err := json.Marshal(result)
	if err != nil {
		return
	}

	bs = append(bs, '\n')

	if !ContextHTTPResponseStarted(reqctx) && flusher == nil {
		w.Header().Set("content-length", strconv.Itoa(len(bs)))
	}

	_, err = w.Write(bs)
	if err != nil {
		return
	}

	if flusher != nil {
		flusher.Flush()
	}

	RequestContext(reqctx).Set(ContextKeyHTTPResponseStarted, true)

	return nil
}
