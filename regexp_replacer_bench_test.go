//go:build go1.24

package main

import "testing"

func BenchmarkRegexpReplacer(b *testing.B) {
	const repetition = 100
	pattern := []byte("hello world!")
	buf := make([]byte, 0, len(pattern)*repetition)
	for i := 0; i < repetition; i++ {
		buf = append(buf, pattern...)
	}
	input := string(buf)
	replacer := NewRegexpReplacer("hello world", "another string")
	for b.Loop() {
		replacer.Replace(input)
	}
}
