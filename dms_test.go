package dms

import "testing"

func TestNew(t *testing.T) {
	sys := New()
	defer sys.Close()
}

type foo struct {
	cb func(Loader)
}

func (f foo) Load(loader Loader) {
	f.cb(loader)
}

func TestLoad(t *testing.T) {
	sys := New()
	defer sys.Close()

	var n int
	var fn func() int
	done := make(chan struct{})
	go sys.Load(foo{
		func(loader Loader) {
			loader.Require("x", &n)
			loader.Require("fn", &fn)
			close(done)
		},
	})
	sys.Load(foo{
		func(loader Loader) {
			loader.Provide("x", 42)
			loader.Provide("fn", func() int {
				return 42
			})
		},
	})
	<-done
	if n != 42 {
		t.Fail()
	}
	if fn == nil || fn() != 42 {
		t.Fail()
	}
}
