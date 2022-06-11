package wsgraphql

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
	"github.com/stretchr/testify/assert"
)

type testWrapper struct {
	*websocket.Upgrader
}

// Upgrade implementation
func (g testWrapper) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (Conn, error) {
	return g.Upgrader.Upgrade(w, r, responseHeader)
}

func testNewSchema(t *testing.T) graphql.Schema {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:       "QueryRoot",
			Interfaces: nil,
			Fields: graphql.Fields{
				"getFoo": &graphql.Field{
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return 123, nil
					},
				},
			},
		}),
		Mutation: graphql.NewObject(graphql.ObjectConfig{
			Name:       "MutationRoot",
			Interfaces: nil,
			Fields: graphql.Fields{
				"setFoo": &graphql.Field{
					Args: graphql.FieldConfigArgument{
						"value": &graphql.ArgumentConfig{
							Type: graphql.Int,
						},
					},
					Type: graphql.Boolean,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						_, ok := p.Args["value"].(int)

						return ok, nil
					},
				},
			},
		}),
		Subscription: graphql.NewObject(graphql.ObjectConfig{
			Name:       "SubscriptionRoot",
			Interfaces: nil,
			Fields: graphql.Fields{
				"fooUpdates": &graphql.Field{
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return p.Source, nil
					},
					Subscribe: func(p graphql.ResolveParams) (interface{}, error) {
						ch := make(chan interface{}, 3)

						ch <- 1
						ch <- 2
						ch <- 3

						close(ch)

						return ch, nil
					},
				},
				"forever": &graphql.Field{
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return p.Source, nil
					},
					Subscribe: func(p graphql.ResolveParams) (interface{}, error) {
						ch := make(chan interface{})

						return ch, nil
					},
				},
			},
		}),
	})

	assert.NoError(t, err)
	assert.NotNil(t, schema)

	return schema
}

func testNewServer(t *testing.T, opts ...ServerOption) *httptest.Server {
	opts = append(opts, WithUpgrader(testWrapper{
		Upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			Subprotocols:    []string{WebsocketSubprotocol},
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}))

	server, err := NewServer(testNewSchema(t), nil, opts...)

	assert.NoError(t, err)
	assert.NotNil(t, server)

	return httptest.NewServer(server)
}
