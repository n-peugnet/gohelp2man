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
