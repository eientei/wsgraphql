package wsgraphql

import (
	"encoding/json"
	"net/http"

	"github.com/eientei/wsgraphql/apollows"
	"github.com/eientei/wsgraphql/mutable"
	"github.com/graphql-go/graphql"
)

func (server *serverImpl) servePlainRequest(
	reqctx mutable.Context,
	w http.ResponseWriter,
	r *http.Request,
) (err error) {
	if server.rejectHTTPQueries {
		return errHTTPQueryRejected
	}

	var payload apollows.PayloadOperation

	opctx := mutable.NewMutableContext(reqctx)

	defer func() {
		err = server.callbacks.OnOperationDone(opctx, &payload, err)

		opctx.Cancel()
	}()

	err = server.callbacks.OnConnect(reqctx, nil)
	if err != nil {
		return
	}

	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		return
	}

	params, astdoc, subscription, result := server.parseAST(opctx, &payload)
	if result != nil {
		err = server.callbacks.OnOperation(opctx, &payload)
		if err != nil {
			return err
		}

		err = json.NewEncoder(w).Encode(result)

		return
	}

	err = server.callbacks.OnOperation(opctx, &payload)
	if err != nil {
		return
	}

	var flusher http.Flusher

	var cres chan *graphql.Result

	w.Header().Set("content-type", "application/json")

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

			err = server.writePlainResult(result, w, flusher)
			if err != nil {
				return
			}
		}
	}
}

func (server *serverImpl) writePlainResult(
	result *graphql.Result,
	w http.ResponseWriter,
	flusher http.Flusher,
) (err error) {
	if result == nil {
		return nil
	}

	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		return
	}

	_, err = w.Write([]byte{'\n'})
	if err != nil {
		return
	}

	if flusher != nil {
		flusher.Flush()
	}

	return nil
}
