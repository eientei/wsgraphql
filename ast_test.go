package wsgraphql

import (
	"context"
	"testing"

	"github.com/eientei/wsgraphql/v1/apollows"
	"github.com/eientei/wsgraphql/v1/mutable"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/gqlerrors"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/stretchr/testify/assert"
)

func TestASTParse(t *testing.T) {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:       "QueryRoot",
			Interfaces: nil,
			Fields: graphql.Fields{
				"foo": &graphql.Field{
					Name: "FooType",
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return 123, nil
					},
				},
			},
		}),
	})

	assert.NoError(t, err)
	assert.NotNil(t, schema)

	server, err := NewServer(schema, nil)

	assert.NoError(t, err)
	assert.NotNil(t, server)

	impl, ok := server.(*serverImpl)

	assert.True(t, ok)

	opctx := mutable.NewMutableContext(context.Background())

	params, astdoc, sub, result := impl.parseAST(opctx, &apollows.PayloadOperation{
		Query:         `query { foo }`,
		Variables:     nil,
		OperationName: "",
	})

	assert.Nil(t, result)
	assert.False(t, sub)
	assert.NotNil(t, astdoc)
	assert.NotNil(t, params)
}

func TestASTParseSubscription(t *testing.T) {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:       "QueryRoot",
			Interfaces: nil,
			Fields: graphql.Fields{
				"foo": &graphql.Field{
					Name: "FooType",
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return 123, nil
					},
				},
			},
		}),
		Subscription: graphql.NewObject(graphql.ObjectConfig{
			Name:       "SubscriptionRoot",
			Interfaces: nil,
			Fields: graphql.Fields{
				"foo": &graphql.Field{
					Name: "FooType",
					Type: graphql.Int,
					Subscribe: func(p graphql.ResolveParams) (interface{}, error) {
						return 123, nil
					},
				},
			},
		}),
	})

	assert.NoError(t, err)
	assert.NotNil(t, schema)

	server, err := NewServer(schema, nil)

	assert.NoError(t, err)
	assert.NotNil(t, server)

	impl, ok := server.(*serverImpl)

	assert.True(t, ok)

	opctx := mutable.NewMutableContext(context.Background())

	params, astdoc, sub, result := impl.parseAST(opctx, &apollows.PayloadOperation{
		Query:         `subscription { foo }`,
		Variables:     nil,
		OperationName: "",
	})

	assert.Nil(t, result)
	assert.True(t, sub)
	assert.NotNil(t, astdoc)
	assert.NotNil(t, params)
}

type testExt struct {
	initFn                 func(ctx context.Context, p *graphql.Params) context.Context
	hasResultFn            func() bool
	getResultFn            func(context.Context) interface{}
	parseDidStartFn        func(ctx context.Context) (context.Context, graphql.ParseFinishFunc)
	validationDidStartFn   func(ctx context.Context) (context.Context, graphql.ValidationFinishFunc)
	executionDidStartFn    func(ctx context.Context) (context.Context, graphql.ExecutionFinishFunc)
	resolveFieldDidStartFn func(
		ctx context.Context,
		i *graphql.ResolveInfo,
	) (context.Context, graphql.ResolveFieldFinishFunc)
	name string
}

func (t *testExt) Init(ctx context.Context, p *graphql.Params) context.Context {
	return t.initFn(ctx, p)
}

func (t *testExt) Name() string {
	return t.name
}

func (t *testExt) HasResult() bool {
	return t.hasResultFn()
}

func (t *testExt) GetResult(ctx context.Context) interface{} {
	return t.getResultFn(ctx)
}

func (t *testExt) ParseDidStart(ctx context.Context) (context.Context, graphql.ParseFinishFunc) {
	return t.parseDidStartFn(ctx)
}

func (t *testExt) ValidationDidStart(ctx context.Context) (context.Context, graphql.ValidationFinishFunc) {
	return t.validationDidStartFn(ctx)
}

func (t *testExt) ExecutionDidStart(ctx context.Context) (context.Context, graphql.ExecutionFinishFunc) {
	return t.executionDidStartFn(ctx)
}

func (t *testExt) ResolveFieldDidStart(
	ctx context.Context,
	i *graphql.ResolveInfo,
) (context.Context, graphql.ResolveFieldFinishFunc) {
	return t.resolveFieldDidStartFn(ctx, i)
}

func testAstParseExtensions(
	t *testing.T,
	f func(ext *testExt),
) (params graphql.Params, astdoc *ast.Document, subscription bool, result *graphql.Result) {
	text := &testExt{
		name: "foo",
		initFn: func(ctx context.Context, p *graphql.Params) context.Context {
			return ctx
		},
		hasResultFn: func() bool {
			return true
		},
		getResultFn: func(ctx context.Context) interface{} {
			return nil
		},
		parseDidStartFn: func(ctx context.Context) (context.Context, graphql.ParseFinishFunc) {
			return ctx, func(err error) {

			}
		},
		validationDidStartFn: func(ctx context.Context) (context.Context, graphql.ValidationFinishFunc) {
			return ctx, func(errors []gqlerrors.FormattedError) {

			}
		},
		executionDidStartFn: func(ctx context.Context) (context.Context, graphql.ExecutionFinishFunc) {
			return ctx, func(result *graphql.Result) {

			}
		},
		resolveFieldDidStartFn: func(
			ctx context.Context,
			i *graphql.ResolveInfo,
		) (context.Context, graphql.ResolveFieldFinishFunc) {
			return ctx, func(i interface{}, err error) {

			}
		},
	}

	f(text)

	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:       "QueryRoot",
			Interfaces: nil,
			Fields: graphql.Fields{
				"foo": &graphql.Field{
					Name: "FooType",
					Type: graphql.Int,
					Resolve: func(p graphql.ResolveParams) (interface{}, error) {
						return 123, nil
					},
				},
			},
		}),
		Extensions: []graphql.Extension{
			text,
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, schema)

	server, err := NewServer(schema, nil)

	assert.NoError(t, err)
	assert.NotNil(t, server)

	impl, ok := server.(*serverImpl)

	assert.True(t, ok)

	opctx := mutable.NewMutableContext(context.Background())

	return impl.parseAST(opctx, &apollows.PayloadOperation{
		Query:         `query { foo }`,
		Variables:     nil,
		OperationName: "",
	})
}

func TestASTParseExtensions(t *testing.T) {
	params, astdoc, sub, result := testAstParseExtensions(t, func(ext *testExt) {

	})

	assert.Nil(t, result)
	assert.False(t, sub)
	assert.NotNil(t, astdoc)
	assert.NotNil(t, params)
}

func TestASTParseExtensionsPanicInit(t *testing.T) {
	params, astdoc, sub, result := testAstParseExtensions(t, func(ext *testExt) {
		ext.initFn = func(ctx context.Context, p *graphql.Params) context.Context {
			panic(1)
		}
	})

	assert.NotNil(t, result)
	assert.False(t, sub)
	assert.Nil(t, astdoc)
	assert.NotNil(t, params)
}

func TestASTParseExtensionsPanicValidation(t *testing.T) {
	params, astdoc, sub, result := testAstParseExtensions(t, func(ext *testExt) {
		ext.validationDidStartFn = func(ctx context.Context) (context.Context, graphql.ValidationFinishFunc) {
			panic(1)
		}
	})

	assert.NotNil(t, result)
	assert.False(t, sub)
	assert.NotNil(t, astdoc)
	assert.NotNil(t, params)
}

func TestASTParseExtensionsPanicValidationCb(t *testing.T) {
	params, astdoc, sub, result := testAstParseExtensions(t, func(ext *testExt) {
		ext.validationDidStartFn = func(ctx context.Context) (context.Context, graphql.ValidationFinishFunc) {
			return ctx, func(errors []gqlerrors.FormattedError) {
				panic(1)
			}
		}
	})

	assert.NotNil(t, result)
	assert.False(t, sub)
	assert.NotNil(t, astdoc)
	assert.NotNil(t, params)
}

func TestASTParseExtensionsPanicParse(t *testing.T) {
	params, astdoc, sub, result := testAstParseExtensions(t, func(ext *testExt) {
		ext.parseDidStartFn = func(ctx context.Context) (context.Context, graphql.ParseFinishFunc) {
			panic(1)
		}
	})

	assert.NotNil(t, result)
	assert.False(t, sub)
	assert.Nil(t, astdoc)
	assert.NotNil(t, params)
}

func TestASTParseExtensionsPanicParseCb(t *testing.T) {
	params, astdoc, sub, result := testAstParseExtensions(t, func(ext *testExt) {
		ext.parseDidStartFn = func(ctx context.Context) (context.Context, graphql.ParseFinishFunc) {
			return ctx, func(err error) {
				panic(1)
			}
		}
	})

	assert.NotNil(t, result)
	assert.False(t, sub)
	assert.NotNil(t, astdoc)
	assert.NotNil(t, params)
}
