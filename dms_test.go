package dms

import (
	"strconv"
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

func TestTypeMismatch(t *testing.T) {
	sys := New()
	defer sys.Close()
	sys.Load(foo{func(loader Loader) {
		loader.Provide("foo", 42)
		var n string
		func() {
			defer func() {
				p := recover()
				if p == nil {
					t.Fatal("should panic")
				}
				if _, ok := p.(ErrTypeMismatch); !ok {
					t.Fatal("should be ErrTypeMismatch")
				}
			}()
			loader.Require("foo", &n)
		}()

		go func() {
			defer func() {
				p := recover()
				if p == nil {
					t.Fatal("should panic")
				}
				if _, ok := p.(ErrTypeMismatch); !ok {
					t.Fatal("should be ErrTypeMismatch")
				}
			}()
			loader.Require("bar", &n)
		}()
		time.Sleep(time.Millisecond * 50) // should be in req queue after sleep
		loader.Provide("bar", 42)
	}})
}

func TestCast(t *testing.T) {
	n := 0
	c := NewCast((*func(int))(nil))
	c.Add(func(i int) {
		n += i
	})
	c.Add(func(i int) {
		n += i * 2
	})
	c.Add(func(i int) {
		n += i * 4
	})
	c.Call(1)
	if n != 7 {
		t.Fail()
	}
}

func TestUnknownCastType(t *testing.T) {
	var c *Cast
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Fatal("should fail")
			}
			if _, ok := p.(ErrUnknownCastType); !ok {
				t.Fatal("should be ErrUnknownCastType")
			}
		}()
		c = NewCast((*func(int, int, int))(nil))
	}()
	AddCastType((*func(int, int, int))(nil), func(fn interface{}, args []interface{}) {
		fn.(func(int, int, int))(args[0].(int), args[1].(int), args[2].(int))
	})
	func() {
		defer func() {
			p := recover()
			if p != nil {
				t.Fail()
			}
		}()
		c = NewCast((*func(int, int, int))(nil))
	}()
	n := 0
	c.Add(func(a, b, c int) {
		n += a + b + c
	})
	c.Call(1, 2, 3)
	if n != 6 {
		t.Fail()
	}
}

func TestBadCastFunc(t *testing.T) {
	c := NewCast((*func())(nil))
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Fatal("should fail")
			}
			if _, ok := p.(ErrBadCastFunc); !ok {
				t.Fatal("should be ErrBadCastFunc")
			}
		}()
		c.Add(func(int) {})
	}()
}

func TestDuration(t *testing.T) {
	d := NewDuration()
	max := 512
	cast := NewCast((*func())(nil))
	res := []int{}
	for i := 0; i < max; i++ {
		i := i
		fn := func() {
			d.Wait(strconv.Itoa(i))
			d.Done(strconv.Itoa(i + 1))
			res = append(res, i)
		}
		cast.Add(fn)
	}
	go cast.Pcall()
	d.Done("0")
	d.Wait(strconv.Itoa(max))
	for i := 0; i < max; i++ {
		if res[i] != i {
			t.Fail()
		}
	}
	d.End()
}

func TestDurationBadEnd(t *testing.T) {
	d := NewDuration()
	go func() {
		d.Wait("foo")
	}()
	time.Sleep(time.Millisecond * 50) // should be waiting after sleep
	func() {
		defer func() {
			p := recover()
			if p == nil {
				t.Fatal("should panic")
			}
			if _, ok := p.(ErrStarvation); !ok {
				t.Fatal("should be ErrStarvation")
			}
		}()
		d.End()
	}()
}
