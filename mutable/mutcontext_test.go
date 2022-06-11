package mutable

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type (
	testFooKeyT struct{}
	testQuxKeyT struct{}
	testDagKeyT struct{}
)

var (
	testFooKey = testFooKeyT{}
	testQuxKey = testQuxKeyT{}
	testDagKey = testDagKeyT{}
)

func TestNew(t *testing.T) {
	mctx := NewMutableContext(context.Background())

	assert.NotNil(t, mctx)
	assert.Nil(t, mctx.Err())

	assert.Equal(t, nil, mctx.Value(testFooKey))
	assert.Equal(t, nil, mctx.Value(testQuxKey))
	assert.Equal(t, nil, mctx.Value(testDagKey))

	mctx.Set(testFooKey, "bar")
	mctx.Set(testQuxKey, "baz")

	assert.Equal(t, "bar", mctx.Value(testFooKey))
	assert.Equal(t, "baz", mctx.Value(testQuxKey))
	assert.Equal(t, nil, mctx.Value(testDagKey))
}

func TestParent(t *testing.T) {
	mctx := NewMutableContext(context.WithValue(context.Background(), testFooKey, "123"))

	assert.NotNil(t, mctx)
	assert.Nil(t, mctx.Err())

	assert.Equal(t, "123", mctx.Value(testFooKey))
	assert.Equal(t, nil, mctx.Value(testQuxKey))
	assert.Equal(t, nil, mctx.Value(testDagKey))

	mctx.Set(testFooKey, "bar")
	mctx.Set(testQuxKey, "baz")

	assert.Equal(t, "bar", mctx.Value(testFooKey))
	assert.Equal(t, "baz", mctx.Value(testQuxKey))
	assert.Equal(t, nil, mctx.Value(testDagKey))
}

func TestCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	mctx := NewMutableContext(ctx)

	assert.Nil(t, mctx.Err())

	cancel()

	assert.Error(t, mctx.Err())
}
