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

func BenchmarkPcallx1(b *testing.B) {
	c := NewCast((*func(int))(nil))
	c.Add(func(int) {})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Pcall(0)
	}
}

func BenchmarkPcallx4(b *testing.B) {
	c := NewCast((*func(int))(nil))
	for i := 0; i < 4; i++ {
		c.Add(func(int) {})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Pcall(0)
	}
}

func BenchmarkPcallx16(b *testing.B) {
	c := NewCast((*func(int))(nil))
	for i := 0; i < 16; i++ {
		c.Add(func(int) {})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.Pcall(0)
	}
}
