package mutcontext

import (
	"context"
	"testing"
)

func TestCreateNew(t *testing.T) {
	parent := context.Background()
	ctx := CreateNew(parent)
	m := ctx.(*mutableContext)
	if m.Context != parent {
		t.Error("invalid context")
	}

	_, ok := m.Deadline()
	if ok {
		t.Error("unexpected deadline")
	}

	if m.Done() != nil {
		t.Error("unexpected done", m.Done())
	}

	if m.Err() != nil {
		t.Error("unexpected err", m.Err())
	}

	if v := m.Value("unexpected"); v != nil {
		t.Error("unexpected value", v)
	}
	m.Set("expected", "foobar")
	if v := m.Value("expected"); v != "foobar" {
		t.Error("unexpected value", v)
	}
	called := false
	m.SetCleanup(func() {
		called = true
	})
	err := m.Cancel()
	if err != ErrNoCancel {
		t.Error("incorrect error", err)
	}
	if !called {
		t.Error("cleanup was not called")
	}
	if !m.Completed() {
		t.Error("cancelled flag not set")
	}
}

func TestCreateNewCancel(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	ctx := CreateNewCancel(parent, cancel)
	m := ctx.(*mutableContext)
	if m.Context != parent {
		t.Error("invalid context")
	}
	if m.CancelFunc == nil {
		t.Error("invalid cancel func")
	}
	err := m.Cancel()
	if err != nil {
		t.Error("unexpected error", err)
	}
}
