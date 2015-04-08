package dms

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func init() {
	var seed int64
	binary.Read(crand.Reader, binary.LittleEndian, &seed)
	rand.Seed(seed)
}

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

type Funcs []func()

func (s Funcs) Shuffle() {
	for i := len(s) - 1; i >= 1; i-- {
		j := rand.Intn(i + 1)
		s[i], s[j] = s[j], s[i]
	}
}

func TestDuration(t *testing.T) {
	d := NewDuration()
	max := 512
	fns := []func(){}
	res := []int{}
	for i := 0; i < max; i++ {
		i := i
		fn := func() {
			d.Wait(strconv.Itoa(i))
			d.Done(strconv.Itoa(i + 1))
			res = append(res, i)
		}
		fns = append(fns, fn)
	}
	Funcs(fns).Shuffle()
	for _, fn := range fns {
		go fn()
	}
	d.Done("0")
	d.Wait(strconv.Itoa(max))
	for i := 0; i < max; i++ {
		if res[i] != i {
			t.Fail()
		}
	}
	d.End()
}
