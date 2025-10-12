package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseFlag(t *testing.T) {
	cases := []struct {
		name     string
		val      string
		expected *Flag
	}{
		{
			"empty string",
			"",
			nil,
		},
		{
			"synopsis",
			"Usage of compose-spec:",
			nil,
		},
		{
			"simple short",
			"  -h	Show help and exit.",
			&Flag{"h", "", "Show help and exit."},
		},
		{
			"simple long",
			`  -help
    	Show help and exit.`,
			&Flag{"help", "", "Show help and exit."},
		},
		{
			"simple arg",
			`  -fmt string
    	Output format (yaml|json). (default "yaml")`,
			&Flag{"fmt", "string", `Output format (yaml|json). (default "yaml")`},
		},
		{
			"kebab case",
			`  -kebab-case
    	Flag using kebab case.`,
			&Flag{"kebab-case", "", "Flag using kebab case."},
		},
		{
			"single digit",
			"  -6	Use IPv6 protocol.",
			&Flag{"6", "", "Use IPv6 protocol."},
		},
		{
			"short with custom arg",
			`  -t V
    	Use V as test. (default "test")`,
			&Flag{"t", "V", `Use V as test. (default "test")`},
		},
		{
			"custom arg with space",
			`  -test V V
    	Use V V as test. (default "test")
`,
			&Flag{"test", "V V", `Use V V as test. (default "test")`},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			help := NewHelp("test", strings.NewReader(c.val))
			help.scanner.Scan()
			f := help.parseFlag()
			if !reflect.DeepEqual(c.expected, f) {
				t.Fatalf("expected:\n%v\ngot:\n%v", c.expected, f)
			}
		})
	}
}
