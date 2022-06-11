Complete server example, with query, mutation and subscriptions.

Running
-------

```go
go run main.go -addr :8080
```

GraphQL endpoint will be available at http://127.0.0.1:8080/query 

Navigate to playground on http://127.0.0.1:8080

And try following queries:

```graphql
subscription {
  fooUpdates
}
```

Then in new tab(s)

```graphql
query {
  getFoo
}
```

```graphql
mutation {
  setFoo(value: 123)
}
```

```graphql
query {
  getFoo
}
```
