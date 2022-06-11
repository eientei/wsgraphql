// Package mutable provides v1.mutable context, that can store multiple values and be updated after creation
package mutable

import (
	"context"
	"sync"
)

// Context interface, provides additional Set method to change values of the context after creation
type Context interface {
	context.Context
	Set(key, value interface{})
	Cancel()
}

type mutableContext struct {
	context.Context
	values map[interface{}]interface{}
	cancel context.CancelFunc
	mutex  sync.RWMutex
}

func (mctx *mutableContext) Set(key, value interface{}) {
	mctx.mutex.Lock()

	mctx.values[key] = value

	mctx.mutex.Unlock()
}

func (mctx *mutableContext) Value(key interface{}) (res interface{}) {
	var ok bool

	mctx.mutex.RLock()

	res, ok = mctx.values[key]

	mctx.mutex.RUnlock()

	if !ok {
		res = mctx.Context.Value(key)
	}

	return
}

func (mctx *mutableContext) Cancel() {
	mctx.cancel()
}

// NewMutableContext returns new Context instance
func NewMutableContext(parent context.Context) Context {
	ctx, cancel := context.WithCancel(parent)

	return &mutableContext{
		Context: ctx,
		values:  make(map[interface{}]interface{}),
		cancel:  cancel,
	}
}
