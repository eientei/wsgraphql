[![Go Doc Reference](https://godoc.org/github.com/eientei/wsgraphql?status.svg)](https://godoc.org/github.com/eientei/wsgraphql)
[![Go Report Card](https://goreportcard.com/badge/github.com/eientei/wsgraphql)](https://goreportcard.com/report/github.com/eientei/wsgraphql)


An implementation of
[apollo graphql](https://github.com/apollographql/subscriptions-transport-ws/blob/master/PROTOCOL.md)
websocket protocol for
[graphql-go](https://github.com/graphql-go/graphql).

Inspired by [graphqlws](https://github.com/functionalfoundry/graphqlws)

Key features:

- Subscription support
- Callbacks at every stage of communication process for easy customization 
- Supports both websockets and plain http queries
- [Mutable context](https://godoc.org/github.com/eientei/wsgraphql/mutable) allowing to keep request-scoped 
  connection/authentication data and operation-scoped state
