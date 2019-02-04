[![](https://godoc.org/github.com/eientei/wsgraphql?status.svg)](https://godoc.org/github.com/eientei/wsgraphql)

Basically an implementation of [apollo graphql](https://github.com/apollographql/subscriptions-transport-ws/blob/master/PROTOCOL.md) websocket protocol for [graphql-go](https://github.com/graphql-go/graphql).

You might want to also to checkout [graphqlws](https://github.com/functionalfoundry/graphqlws), which has inspired this implementation.

Key features:

- Subscription support
- Callbacks at every stage of communication process for easy customization 
- Supports both websockets and plain http queries (with exception of continuing subscriptions)
- Mutable context allowing to keep global-scoped connection/authentication data and subscription-scoped state

Feedback/PR is surely welcome.