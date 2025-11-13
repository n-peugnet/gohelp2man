// This file is part of gohelp2man.
//
// Copyright (C) 2025  Nicolas Peugnet <nicolas@club1.fr>
//
// gohelp2man is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; either version 2
// of the License, or (at your option) any later version.
//
// gohelp2man is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program; if not, see <https://www.gnu.org/licenses/>.

//go:build go1.24

package main

import (
	"regexp"
	"strings"
	"testing"
)

func benchmarkLargeInput() string {
	const repetition = 100
	pattern := []byte("hello world!")
	buf := make([]byte, 0, len(pattern)*repetition)
	for i := 0; i < repetition; i++ {
		buf = append(buf, pattern...)
	}
	return string(buf)
}

func benchmarkLargeExpected() string {
	const repetition = 100
	pattern := []byte("another string!")
	buf := make([]byte, 0, len(pattern)*repetition)
	for i := 0; i < repetition; i++ {
		buf = append(buf, pattern...)
	}
	return string(buf)
}

func BenchmarkStringReplacerBaseline(b *testing.B) {
	input := benchmarkLargeInput()
	replacer := strings.NewReplacer("hello", "another", "world", "string")
	var actual string
	for b.Loop() {
		actual = replacer.Replace(input)
	}
	if actual != benchmarkLargeExpected() {
		b.Error("unexpected results")
	}
}

func BenchmarkNaiveRegexReplacer(b *testing.B) {
	input := benchmarkLargeInput()
	reHello := regexp.MustCompile("hello")
	reWorld := regexp.MustCompile("world")
	var actual string
	for b.Loop() {
		actual = reHello.ReplaceAllString(input, "another")
		actual = reWorld.ReplaceAllString(actual, "string")
	}
	if actual != benchmarkLargeExpected() {
		b.Error("unexpected results")
	}
}

func BenchmarkRegexpReplacer(b *testing.B) {
	input := benchmarkLargeInput()
	replacer := NewRegexpReplacer("hello", "another", "world", "string")
	var actual string
	for b.Loop() {
		actual = replacer.Replace(input)
	}
	if actual != benchmarkLargeExpected() {
		b.Error("unexpected results")
	}
}
