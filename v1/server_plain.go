package wsgraphql

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/graphql-go/graphql"
)

type resultError struct {
	*graphql.Result
}

func (r resultError) Error() string {
	bs, _ := json.Marshal(r.Result)

	return string(bs)
}

func (server *serverImpl) servePlainRequest(
	reqctx mutable.Context,
	w http.ResponseWriter,
	r *http.Request,
) (err error) {
	if server.rejectHTTPQueries {
		return errHTTPQueryRejected
	}

	err = server.callbacks.OnConnect(reqctx, nil)
	if err != nil {
		return
	}

	var payload apollows.PayloadOperation

	opctx := mutable.NewMutableContext(reqctx)

	defer opctx.Cancel()

	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return
	}

	defer func() {
		err = server.callbacks.OnOperationDone(opctx, &payload, err)
	}()

	err = server.callbacks.OnOperation(opctx, &payload)
	if err != nil {
		return err
	}

	params, astdoc, subscription, result := server.parseAST(opctx, &payload)

	err = server.callbacks.OnOperationValidation(opctx, &payload, result)
	if err != nil {
		return err
	}

	if result != nil {
		return resultError{Result: result}
	}

	w.Header().Set("content-type", "application/json")

	var (
		cres    chan *graphql.Result
		flusher http.Flusher
	)

	if subscription {
		flusher, _ = w.(http.Flusher)
		w.Header().Set("x-content-type-options", "nosniff")
		w.Header().Set("connection", "keep-alive")

		cres = graphql.ExecuteSubscription(graphql.ExecuteParams{
			Schema:        server.schema,
			Root:          server.rootObject,
			AST:           astdoc,
			OperationName: payload.OperationName,
			Args:          payload.Variables,
			Context:       params.Context,
		})
	} else {
		cres = make(chan *graphql.Result, 1)
		cres <- graphql.Execute(graphql.ExecuteParams{
			Schema:        server.schema,
			Root:          server.rootObject,
			AST:           astdoc,
			OperationName: payload.OperationName,
			Args:          payload.Variables,
			Context:       params.Context,
		})
		close(cres)
	}

	var ok bool

	for {
		select {
		case <-params.Context.Done():
			return params.Context.Err()
		case result, ok = <-cres:
			if !ok {
				return
			}

			err = server.callbacks.OnOperationResult(opctx, &payload, result)
			if err != nil {
				return err
			}

			err = server.writePlainResult(reqctx, result, w, flusher)
			if err != nil {
				return
			}
		}
	}
}

func (server *serverImpl) writePlainResult(
	reqctx mutable.Context,
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

	reqctx.Set(ContextKeyHTTPResponseStarted, true)

	return nil
}
