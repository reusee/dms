package dms

import "testing"

func BenchmarkRequire(b *testing.B) {
	sys := New()
	sys.Load(foo{func(loader Loader) {
		loader.Provide("x", 42)
	}})
	sys.Load(foo{func(loader Loader) {
		var n int
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			loader.Require("x", &n)
		}
		if n != 42 {
			b.Fail()
		}
	}})
}

func BenchmarkCast(b *testing.B) {
	c := NewCast((*func(int))(nil))
	c.Add(func(int) {})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Call(0)
	}
}
