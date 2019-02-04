// Mutable context, allows for easy setting additional values shared across all context and it's children
package mutcontext

import (
	"context"
	"errors"
	"time"
)

// returned if no cancel func present
var ErrNoCancel = errors.New("no cancel func")

// cleanup closure
type FuncCleanup func()

// Mutable context interface, allows easy setting values and keeping cancel() function
type MutableContext interface {
	context.Context
	Set(key, value interface{})
	Cancel() error
	SetCleanup(cleanup FuncCleanup)
	Complete()
	WasCompleted() bool
}

// Basic interface implementation, also uses map instead of delegates for efficiency for many keys
type mutableContext struct {
	Context     context.Context
	CancelFunc  context.CancelFunc
	CleanupFunc FuncCleanup
	Values      map[interface{}]interface{}
	IsComplete  bool
}

// Pass-through to parent context
func (ctx *mutableContext) Deadline() (deadline time.Time, ok bool) {
	return ctx.Context.Deadline()
}

// Pass-through to parent context
func (ctx *mutableContext) Done() <-chan struct{} {
	return ctx.Context.Done()
}

// Pass-through to parent context
func (ctx *mutableContext) Err() error {
	return ctx.Context.Err()
}

// If contained in local map, use that, otherwise pass-through to parent context
func (ctx *mutableContext) Value(key interface{}) interface{} {
	if v, ok := ctx.Values[key]; ok {
		return v
	}
	return ctx.Context.Value(key)
}

// Put value in local map
func (ctx *mutableContext) Set(key, value interface{}) {
	ctx.Values[key] = value
}

// If have cancel() function, use that, otherwise return error
func (ctx *mutableContext) Cancel() error {
	if ctx.IsComplete {
		return ctx.Err()
	}
	ctx.Complete()
	if ctx.CancelFunc == nil {
		return ErrNoCancel
	}
	ctx.CancelFunc()
	return nil
}

// assigned cleanup function
func (ctx *mutableContext) SetCleanup(cleanup FuncCleanup) {
	ctx.CleanupFunc = cleanup
}

// complete context gracefully
func (ctx *mutableContext) Complete() {
	if ctx.IsComplete {
		return
	}
	ctx.IsComplete = true
	if ctx.CleanupFunc != nil {
		ctx.CleanupFunc()
	}
}

// indicates context was completed normally
func (ctx *mutableContext) WasCompleted() bool {
	return ctx.IsComplete
}

// Constructor without cancel() function, will make ctx.Cancel() != nil
func CreateNew(ctx context.Context) MutableContext {
	return &mutableContext{
		Context: ctx,
		Values:  make(map[interface{}]interface{}),
	}
}

// Constructor with cancel() function
func CreateNewCancel(ctx context.Context, cancel context.CancelFunc) MutableContext {
	return &mutableContext{
		Context:    ctx,
		CancelFunc: cancel,
		Values:     make(map[interface{}]interface{}),
	}
}
