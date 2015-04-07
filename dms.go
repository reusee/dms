package dms

import (
	"reflect"
	"sync"
	"time"
)

var (
	MaxResolveTime = time.Second * 8
)

type Mod interface {
	Load(loader Loader)
}

type Loader struct {
	reqChan chan req
	proChan chan pro
}

type Sys struct {
	loader Loader
	closed chan struct{}
}

type req struct {
	name string
	p    interface{}
	done chan struct{}
}

type pro struct {
	name string
	v    interface{}
}

func New() *Sys {
	closed := make(chan struct{})

	// start keeper
	reqChan := make(chan req)
	proChan := make(chan pro)
	go func() {
		keep := make(map[string]reflect.Value)
		reqs := make(map[string][]req)
		for {
			select {
			case r := <-reqChan:
				if v, ok := keep[r.name]; ok {
					reflect.ValueOf(r.p).Elem().Set(v)
					r.done <- struct{}{}
				} else {
					reqs[r.name] = append(reqs[r.name], r)
				}
			case p := <-proChan:
				v := reflect.ValueOf(p.v)
				keep[p.name] = v
				for _, r := range reqs[p.name] {
					reflect.ValueOf(r.p).Elem().Set(v)
					r.done <- struct{}{}
				}
				reqs[p.name] = reqs[p.name][0:0]
			case <-closed:
				return
			}
		}
	}()

	return &Sys{
		loader: Loader{
			reqChan: reqChan,
			proChan: proChan,
		},
		closed: closed,
	}
}

func (s *Sys) Close() {
	close(s.closed)
}

func (s *Sys) Load(mod Mod) {
	mod.Load(s.loader)
}

var sigChanPool = sync.Pool{
	New: func() interface{} {
		return make(chan struct{})
	},
}

func (l Loader) Require(name string, p interface{}) {
	done := sigChanPool.Get().(chan struct{})
	l.reqChan <- req{
		name: name,
		p:    p,
		done: done,
	}
	select {
	case <-done:
	case <-time.After(MaxResolveTime):
		panic(ErrNotProvided{name})
	}
	sigChanPool.Put(done)
}

func (l Loader) Provide(name string, v interface{}) {
	l.proChan <- pro{
		name: name,
		v:    v,
	}
}

type Cast struct {
	fns  []interface{}
	what reflect.Type
}

func NewCast(castType interface{}) *Cast {
	what := reflect.TypeOf(castType).Elem()
	if _, ok := castHandlers[what]; !ok {
		panic(ErrUnknownCastType{what})
	}
	return &Cast{
		what: what,
	}
}

var castHandlers = map[reflect.Type]func(fns, args []interface{}){
	reflect.TypeOf((*func(int))(nil)).Elem(): func(fns, args []interface{}) {
		for _, fn := range fns {
			fn.(func(int))(args[0].(int))
		}
	},
}

func AddCastType(p interface{}, handler func(fns, args []interface{})) {
	castHandlers[reflect.TypeOf(p).Elem()] = handler
}

func (c *Cast) Call(args ...interface{}) {
	castHandlers[c.what](c.fns, args)
}

func (c *Cast) Add(fn interface{}) {
	if reflect.TypeOf(fn) != c.what {
		panic(ErrBadCastFunc{fn})
	}
	c.fns = append(c.fns, fn)
}
