package dms

import (
	"testing"
	"time"
)

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

	var n, m int
	var fn func() int
	done := make(chan bool)
	go sys.Load(foo{
		func(loader Loader) {
			loader.Require("x", &n)
			loader.Require("fn", &fn)
			loader.Provide("y", 42)
			done <- true
		},
	})
	go sys.Load(foo{
		func(loader Loader) {
			loader.Provide("x", 42)
			loader.Provide("fn", func() int {
				return 42
			})
			loader.Require("y", &m)
			done <- true
		},
	})
	<-done
	<-done
	if n != 42 {
		t.Fail()
	}
	if m != 42 {
		t.Fail()
	}
	if fn == nil || fn() != 42 {
		t.Fail()
	}
}

func TestNotProvided(t *testing.T) {
	old := MaxResolveTime
	MaxResolveTime = time.Millisecond * 50
	sys := New()
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Fatal("should panic")
			}
			if e, ok := p.(ErrNotProvided); !ok {
				t.Fatal("should be ErrNotProvided")
			} else {
				if e.What != "foo" {
					t.Fatal("should be foo")
				}
			}
		}()
		sys.Load(foo{func(loader Loader) {
			var x int
			loader.Require("foo", &x)
		}})
	}()
	MaxResolveTime = old
}
