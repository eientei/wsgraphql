[![Go Doc Reference](https://godoc.org/github.com/eientei/wsgraphql/v1?status.svg)](https://godoc.org/github.com/eientei/wsgraphql/v1)
[![Go Report Card](https://goreportcard.com/badge/github.com/eientei/wsgraphql)](https://goreportcard.com/report/github.com/eientei/wsgraphql)
[![Maintainability](https://api.codeclimate.com/v1/badges/c626b5f2399b044bdebf/maintainability)](https://codeclimate.com/github/eientei/wsgraphql)
[![Test Coverage](https://api.codeclimate.com/v1/badges/c626b5f2399b044bdebf/test_coverage)](https://codeclimate.com/github/eientei/wsgraphql)

An implementation of
[apollo graphql](https://github.com/apollographql/subscriptions-transport-ws/blob/master/PROTOCOL.md)
websocket protocol for
[graphql-go](https://github.com/graphql-go/graphql).

Inspired by [graphqlws](https://github.com/functionalfoundry/graphqlws)

Key features:

- Subscription support
- Callbacks at every stage of communication process for easy customization 
- Supports both websockets and plain http queries, with http chunked response for plain http subscriptions
- [Mutable context](https://godoc.org/github.com/eientei/wsgraphql/v1/mutable) allowing to keep request-scoped 
  connection/authentication data and operation-scoped state

Usage
-----

Assuming [gorilla websocket](https://github.com/gorilla/websocket) upgrader

```go
import (
	"net/http"

	"github.com/eientei/wsgraphql/v1"
	"github.com/eientei/wsgraphql/v1/compat/gorillaws"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
)
```

```go
schema, err := graphql.NewSchema(...)
if err != nil {
	panic(err)
}

srv, err := wsgraphql.NewServer(
	schema,
	nil,
	wsgraphql.WithUpgrader(gorillaws.Wrap(&websocket.Upgrader{
		Subprotocols: []string{wsgraphql.WebsocketSubprotocol},
	})),
)
if err != nil {
	panic(err)
}

http.Handle("/query", srv)

err = http.ListenAndServe(":8080", nil)
if err != nil {
	panic(err)
}
```

Examples
--------

See [/v1/examples](/v1/examples)
- [minimal](/v1/examples/minimal) server setup
- [simpleserver](/v1/examples/simpleserver) complete example with subscriptions, mutations and queries
