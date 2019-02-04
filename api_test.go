package wsgraphql

import (
	"bytes"
	"errors"
	"net/http/httptest"

	"github.com/eientei/wsgraphql/server"

	"testing"

	"github.com/graphql-go/graphql"
)

func TestNewServer_noschema(t *testing.T) {
	_, err := NewServer(Config{})
	if err == nil {
		t.Error("schema must be asserted")
	}
	if err != ErrSchemaRequired {
		t.Error("unexpected error", err)
	}
}

func TestNewServer_minimal_subscription(t *testing.T) {
	schema, err := graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"foo": &graphql.Field{
					Type: graphql.Int,
				},
			},
		}),
	})
	if err != nil {
		t.Error("unexpected schema error", err)
	}
	serv, err := NewServer(Config{
		Schema: &schema,
	})
	if err != nil {
		t.Error("unexpected error", err)
	}
	s := serv.(*server.Server)
	rec := httptest.NewRecorder()
	s.OnPlainFail(nil, nil, rec, errors.New("foobar"))
	if !bytes.Equal(rec.Body.Bytes(), []byte("foobar")) {
		t.Error("invalid fail writer")
	}
}
