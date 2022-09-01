package wsgraphql

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
	"github.com/stretchr/testify/assert"
)

type testWrapper struct {
	*websocket.Upgrader
}

type testConn struct {
	*websocket.Conn
}

func (conn testConn) Close(code int, message string) error {
	origerr := conn.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(code, message))

	err := conn.Conn.Close()
	if err == nil {
		err = origerr
	}

	return err
}

// Upgrade implementation
func (g testWrapper) Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (Conn, error) {
	c, err := g.Upgrader.Upgrade(w, r, responseHeader)
	if err != nil {
		return nil, err
	}

	return testConn{
		Conn: c,
	}, nil
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
				"getError": &graphql.Field{
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return nil, errors.New("someerr")
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

func testNewServerProtocols(t *testing.T, protocols []apollows.Protocol, opts ...ServerOption) *httptest.Server {
	var strprotocols []string

	for _, p := range protocols {
		strprotocols = append(strprotocols, p.String())
		opts = append(opts, WithProtocol(p))
	}

	opts = append(opts, WithUpgrader(testWrapper{
		Upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			Subprotocols:    strprotocols,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}))

	server, err := NewServer(testNewSchema(t), opts...)

	assert.NoError(t, err)
	assert.NotNil(t, server)

	return httptest.NewServer(server)
}

func testNewServer(t *testing.T, protocol apollows.Protocol, opts ...ServerOption) *httptest.Server {
	return testNewServerProtocols(t, []apollows.Protocol{protocol}, opts...)
}
