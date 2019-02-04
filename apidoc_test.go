package wsgraphql

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/eientei/wsgraphql/mutcontext"
	"github.com/graphql-go/graphql"
)

func Example() {
	value := 0
	var changechan chan int

	query := graphql.NewObject(graphql.ObjectConfig{
		Name: "TestQuery",
		Fields: graphql.Fields{
			"get": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
				Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
					return value, nil
				},
			},
		},
	})

	mutation := graphql.NewObject(graphql.ObjectConfig{
		Name: "TestMutation",
		Fields: graphql.Fields{
			"set": &graphql.Field{
				Type: graphql.Int,
				Args: graphql.FieldConfigArgument{
					"value": &graphql.ArgumentConfig{
						Type: graphql.NewNonNull(graphql.Int),
					},
				},
				Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
					value = p.Args["value"].(int)
					if changechan != nil {
						changechan <- value
					}
					return nil, nil
				},
			},
			"stop": &graphql.Field{
				Type: graphql.Int,
				Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
					if changechan != nil {
						close(changechan)
						changechan = nil
					}
					return nil, nil
				},
			},
		},
	})

	subscription := graphql.NewObject(graphql.ObjectConfig{
		Name: "TestSubscription",
		Fields: graphql.Fields{
			"watch": &graphql.Field{
				Type: graphql.NewNonNull(graphql.Int),
				Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
					ctx := p.Context.(mutcontext.MutableContext)
					c := ctx.Value("ch")
					if c == nil {
						newc := make(chan int)
						ctx.Set("ch", newc)
						ctx.SetCleanup(func() {
							c := ctx.Value("ch")
							if c != nil {
								close(c.(chan int))
							}
						})
						c = newc
						changechan = newc
					}
					ch := c.(chan int)
					v, ok := <-ch
					if !ok {
						ctx.Set("ch", nil)
						_ = ctx.Cancel()
					}
					return v, nil
				},
			},
			"countdown": &graphql.Field{
				Type: graphql.Int,
				Args: graphql.FieldConfigArgument{
					"value": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 0,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					ctx := p.Context.(mutcontext.MutableContext)

					iter := ctx.Value("iter")
					if iter == nil {
						v, ok := p.Args["value"].(int)
						if !ok {
							return nil, nil
						}
						ctx.Set("iter", v)
						iter = v
					}

					i := iter.(int)
					if i <= 0 {
						ctx.Complete()
						return 0, nil
					}

					ctx.Set("iter", i-1)
					return i, nil
				},
			},
		},
	})

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "RootQuery",
			Fields: graphql.Fields{
				"test": &graphql.Field{
					Type: query,
					Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
						return query, nil
					},
				},
			},
		}),
		Mutation: graphql.NewObject(graphql.ObjectConfig{
			Name: "RootMutation",
			Fields: graphql.Fields{
				"test": &graphql.Field{
					Type: mutation,
					Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
						return mutation, nil
					},
				},
			},
		}),
		Subscription: graphql.NewObject(graphql.ObjectConfig{
			Name: "RootSubscription",
			Fields: graphql.Fields{
				"test": &graphql.Field{
					Type: subscription,
					Resolve: func(p graphql.ResolveParams) (i interface{}, e error) {
						return subscription, nil
					},
				},
			},
		}),
	})

	server, err := NewServer(Config{
		Schema: &schema,
	})
	if err != nil {
		panic(err)
	}
	http.Handle("/graphql", server)

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println(err)
	}
}

func Example_authentication() {
	var schema graphql.Schema
	server, err := NewServer(Config{
		Schema: &schema,
		OnPlainInit: func(globalctx mutcontext.MutableContext, r *http.Request, w http.ResponseWriter) {
			remote := r.RemoteAddr
			idx := strings.LastIndex(remote, ":")
			globalctx.Set("remote", remote[:idx])
		},
		OnConnect: func(globalctx mutcontext.MutableContext, parameters interface{}) error {
			mapparams := parameters.(map[string]interface{})
			v, ok := mapparams["token"].(string)
			if !ok {
				return errors.New("invalid token type")
			}
			remoteip := globalctx.Value("remote").(string)
			type User struct {
				Name string
			}
			authenticateUser := func(token, remote string) *User {
				if token == "123" {
					return &User{Name: "admin"}
				}
				return nil
			}

			user := authenticateUser(v, remoteip)
			globalctx.Set("user", user)
			return nil
		},
	})
	if err != nil {
		panic(err)
	}
	http.Handle("/graphql", server)

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println(err)
	}
}
