package main

import (
	"flag"
	"net/http"

	"github.com/eientei/wsgraphql/v1"
	"github.com/eientei/wsgraphql/v1/compat/gorillaws"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
)

func main() {
	var addr string

	flag.StringVar(&addr, "addr", ":8080", "Address to listen on")
	flag.Parse()

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
		wsgraphql.WithProtocol(wsgraphql.WebsocketSubprotocolGraphqlTransportWS),
		wsgraphql.WithUpgrader(gorillaws.Wrap(&websocket.Upgrader{
			Subprotocols: []string{string(wsgraphql.WebsocketSubprotocolGraphqlTransportWS)},
		})),
	)
	if err != nil {
		panic(err)
	}

	http.Handle("/query", srv)

	err = http.ListenAndServe(addr, nil)
	if err != nil {
		panic(err)
	}
}
