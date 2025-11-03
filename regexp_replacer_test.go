package main_test

import (
	main "github.com/n-peugnet/gohelp2man"
	"testing"
)

func TestRegexpReplacer(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		repls    []string
		expected string
	}{
		{
			name:     "basic",
			input:    "hello basic test",
			repls:    []string{"hello", "world", "test", "bar"},
			expected: "world basic bar",
		},
		{
			name:     "simple regex",
			input:    "multiple       spaces",
			repls:    []string{"multiple", "single", "\\s+", " ", "s\\b", ""},
			expected: "single space",
		},
		{
			name:     "regex with ^",
			input:    ". Leading dot.",
			repls:    []string{`^\.`, "*"},
			expected: "* Leading dot.",
		},
		{
			name:     "regex with submatch",
			input:    "use option -help for help",
			repls:    []string{"\\B(-\\w+)\\b", "*${1}*", "help", "fun"},
			expected: "use option *-help* for fun",
		},
		{
			name:     "overlapping first wins",
			input:    "hello hell test",
			repls:    []string{"hello", "world", "hell", "bar"},
			expected: "world bar test",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			replacer := main.NewRegexpReplacer(c.repls...)
			output := replacer.Replace(c.input)
			if output != c.expected {
				t.Logf("input: %q, repls: %q", c.input, c.repls)
				t.Fatalf("expected %q, got %q", c.expected, output)
			}
		})
	}
}
