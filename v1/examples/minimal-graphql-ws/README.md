Minimal server example

Running
-------

```go
go run main.go -addr :8080
```

Graphql endpoint will be available at http://127.0.0.1:8080/query

There is no playground in this example, but you can try following query with graphql client of your choice:

```graphql
query {
  getFoo
}
```
