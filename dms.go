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
	res  chan error
}

type pro struct {
	name string
	v    interface{}
	res  chan error
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
					if target := reflect.ValueOf(r.p).Elem(); target.Type() != v.Type() {
						r.res <- ErrTypeMismatch{v.Type(), target.Type()}
					} else {
						target.Set(v)
						r.res <- nil
					}
				} else {
					reqs[r.name] = append(reqs[r.name], r)
				}
			case p := <-proChan:
				if _, ok := keep[p.name]; ok {
					p.res <- ErrDuplicatedProvision{p.name}
				} else {
					v := reflect.ValueOf(p.v)
					keep[p.name] = v
					for _, r := range reqs[p.name] {
						if target := reflect.ValueOf(r.p).Elem(); target.Type() != v.Type() {
							r.res <- ErrTypeMismatch{v.Type(), target.Type()}
						} else {
							target.Set(v)
							r.res <- nil
						}
					}
					reqs[p.name] = reqs[p.name][0:0]
					p.res <- nil
				}
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

var resChanPool = sync.Pool{
	New: func() interface{} {
		return make(chan error)
	},
}

func (l Loader) Require(name string, p interface{}) {
	res := resChanPool.Get().(chan error)
	l.reqChan <- req{
		name: name,
		p:    p,
		res:  res,
	}
	select {
	case err := <-res:
		if err != nil {
			panic(err)
		}
	case <-time.After(MaxResolveTime):
		panic(ErrNotProvided{name})
	}
	resChanPool.Put(res)
}

func (l Loader) Provide(name string, v interface{}) {
	res := resChanPool.Get().(chan error)
	l.proChan <- pro{
		name: name,
		v:    v,
		res:  res,
	}
	if err := <-res; err != nil {
		panic(err)
	}
	resChanPool.Put(res)
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

var castHandlers = map[reflect.Type]func(fn interface{}, args []interface{}){
	reflect.TypeOf((*func())(nil)).Elem(): func(fn interface{}, args []interface{}) {
		fn.(func())()
	},
	reflect.TypeOf((*func(int))(nil)).Elem(): func(fn interface{}, args []interface{}) {
		fn.(func(int))(args[0].(int))
	},
}

func AddCastType(p interface{}, handler func(fn interface{}, args []interface{})) {
	castHandlers[reflect.TypeOf(p).Elem()] = handler
}

func (c *Cast) Call(args ...interface{}) {
	handler := castHandlers[c.what]
	for _, fn := range c.fns {
		handler(fn, args)
	}
}

func (c *Cast) Pcall(args ...interface{}) {
	handler := castHandlers[c.what]
	wg := new(sync.WaitGroup)
	wg.Add(len(c.fns))
	for _, fn := range c.fns {
		fn := fn
		go func() {
			handler(fn, args)
			wg.Done()
		}()
	}
	wg.Wait()
}

func (c *Cast) Add(fn interface{}) {
	if reflect.TypeOf(fn) != c.what {
		panic(ErrBadCastFunc{fn})
	}
	c.fns = append(c.fns, fn)
}

type Duration struct {
	cond    *sync.Cond
	state   map[string]struct{}
	waiting int
}

func NewDuration() *Duration {
	d := &Duration{
		cond: sync.NewCond(new(sync.Mutex)),
	}
	d.Start()
	return d
}

func (d *Duration) Start() {
	d.state = make(map[string]struct{})
	d.waiting = 0
}

func (d *Duration) Wait(what string) {
	d.cond.L.Lock()
	d.waiting++
	for _, ok := d.state[what]; !ok; _, ok = d.state[what] {
		d.cond.Wait()
	}
	d.waiting--
	d.cond.L.Unlock()
}

func (d *Duration) Done(what string) {
	d.cond.L.Lock()
	d.state[what] = struct{}{}
	d.cond.Broadcast()
	d.cond.L.Unlock()
}

func (d *Duration) End() {
	d.cond.L.Lock()
	if d.waiting != 0 {
		panic(ErrStarvation{})
	}
	d.cond.L.Unlock()
}
