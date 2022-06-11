package main

import (
	"net/http"

	"github.com/eientei/wsgraphql/v1"
	"github.com/eientei/wsgraphql/v1/compat/gorillaws"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
)

func main() {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "QueryRoot",
			Fields: graphql.Fields{
				"getFoo": &graphql.Field{
					Description: "Returns most recent foo value",
					Type:        graphql.NewNonNull(graphql.Int),
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return 123, nil
					},
				},
			},
		}),
	})
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
}
