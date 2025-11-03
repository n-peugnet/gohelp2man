// This file is a part of gohelp2man
//
// Copyright (C) 2025  Nicolas Peugnet <nicolas@club1.fr>
//
// gohelp2man is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; either version 2
// of the License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program; if not, see <https://www.gnu.org/licenses/>.

package main

import (
	"regexp"
	"strings"
)

type RegexpReplacer struct {
	compound *regexp.Regexp
	subexps  []int
	regexps  []*regexp.Regexp
	repls    []string
}

func NewRegexpReplacer(oldnew ...string) *RegexpReplacer {
	if len(oldnew)%2 == 1 {
		panic("RegexpReplacer: odd argument count")
	}
	var (
		subexps []int
		regexps []*regexp.Regexp
		repls   []string
	)
	// Create a compound regex that match all of the "old" values
	buf := []byte{'('}
	for i := 0; i < len(oldnew); i += 2 {
		old := oldnew[i]
		new := oldnew[i+1]
		re := regexp.MustCompile(old)
		if re.Match([]byte{}) {
			panic("RegexpReplacer: regexp matches empty string: " + old)
		}
		regexps = append(regexps, re)
		repls = append(repls, new)
		subexps = append(subexps, re.NumSubexp()+1)
		buf = append(buf, '(')
		buf = append(buf, old[:]...)
		buf = append(buf, ')', '|')
	}
	buf[len(buf)-1] = ')'
	return &RegexpReplacer{
		compound: regexp.MustCompile(string(buf)),
		subexps:  subexps,
		regexps:  regexps,
		repls:    repls,
	}
}

func (rr *RegexpReplacer) Replace(s string) string {
	builder := strings.Builder{}
	pos := 0
	matches := rr.compound.FindAllStringSubmatchIndex(s, -1)
	for pos < len(s) {
		if len(matches) == 0 {
			builder.WriteString(s[pos:])
			break
		}
		submatches := matches[0]
		// Ignore both full match and the first submatch used to create
		// the coumpound regex
		submatch := 4
		for i, subexp := range rr.subexps {
			start := submatches[submatch]
			end := submatches[submatch+1]
			if start != -1 {
				builder.WriteString(s[pos:start])
				re := rr.regexps[i]
				repl := rr.repls[i]
				new := re.ReplaceAllString(s[start:end], repl)
				builder.WriteString(new)
				pos = end
				break
			}
			submatch += subexp * 2
		}
		matches = matches[1:]
	}
	return builder.String()
}
