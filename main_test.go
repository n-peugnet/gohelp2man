package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseUsage(t *testing.T) {
	cases := []struct {
		name  string
		val   string
		usage string
		found bool
	}{
		{
			"empty string",
			"",
			"",
			false,
		},
		{
			"short flag",
			"  -h	Show help and exit.",
			"",
			false,
		},
		{
			"go flag default",
			"Usage of gohelp2man:",
			"gohelp2man",
			true,
		},
		{
			"custom GNU-like",
			"Usage: gohelp2man [OPTION]... EXECUTABLE",
			"gohelp2man [OPTION]... EXECUTABLE",
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			help := NewHelp(strings.NewReader(c.val))
			help.scanner.Scan()
			f, found := help.parseUsage()
			if found != c.found {
				t.Errorf("expected found to be %v, got %v", c.found, found)
			}
			if !reflect.DeepEqual(c.usage, f) {
				t.Fatalf("expected:\n%v\ngot:\n%v", c.usage, f)
			}
		})
	}
}

func TestParseFlag(t *testing.T) {
	cases := []struct {
		name  string
		val   string
		flag  *Flag
		found bool
	}{
		{
			"empty string",
			"",
			nil,
			false,
		},
		{
			"synopsis",
			"Usage of compose-spec:",
			nil,
			false,
		},
		{
			"simple short",
			"  -h	Show help and exit.",
			&Flag{"h", "", "Show help and exit."},
			true,
		},
		{
			"simple long",
			`  -help
    	Show help and exit.`,
			&Flag{"help", "", "Show help and exit."},
			true,
		},
		{
			"simple arg",
			`  -fmt string
    	Output format (yaml|json). (default "yaml")`,
			&Flag{"fmt", "string", `Output format (yaml|json). (default "yaml")`},
			true,
		},
		{
			"kebab case",
			`  -kebab-case
    	Flag using kebab case.`,
			&Flag{"kebab-case", "", "Flag using kebab case."},
			true,
		},
		{
			"single digit",
			"  -6	Use IPv6 protocol.",
			&Flag{"6", "", "Use IPv6 protocol."},
			true,
		},
		{
			"short with custom arg",
			`  -t V
    	Use V as test. (default "test")`,
			&Flag{"t", "V", `Use V as test. (default "test")`},
			true,
		},
		{
			"custom arg with space",
			`  -test V V
    	Use V V as test. (default "test")
`,
			&Flag{"test", "V V", `Use V V as test. (default "test")`},
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			help := NewHelp(strings.NewReader(c.val))
			help.scanner.Scan()
			f, found := help.parseFlag()
			if found != c.found {
				t.Errorf("expected found to be %v, got %v", c.found, found)
			}
			if !reflect.DeepEqual(c.flag, f) {
				t.Fatalf("expected:\n%v\ngot:\n%v", c.flag, f)
			}
		})
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		name string
		val  string
		help *Help
		err  string
	}{
		{
			name: "empty string",
			val:  "",
			help: &Help{},
		},
		{
			name: "description before usage",
			val: `A test help message.

Usage: test [OPTION]... ARG
`,
			help: &Help{
				Usage:       "test [OPTION]... ARG",
				Description: "A test help message.",
			},
		},
		{
			name: "description after usage",
			val: `Usage: test [OPTION]... ARG

A test help message.
`,
			help: &Help{
				Usage:       "test [OPTION]... ARG",
				Description: "A test help message.",
			},
		},
		{
			name: "description after flags",
			val: `Usage: test [OPTION]... ARG
  -h	Show help.

A test help message.
`,
			help: &Help{
				Usage:       "test [OPTION]... ARG",
				Description: "A test help message.",
				Flags:       []*Flag{{"h", "", "Show help."}},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			help := NewHelp(strings.NewReader(c.val))
			err := help.parse()
			if c.err != "" {
				if !strings.Contains(err.Error(), c.err) {
					t.Fatalf("expected error to contain %q, got %q", c.err, err)
				}
				return
			}
			if help.Usage != c.help.Usage {
				t.Errorf("expected usage:\n%v\ngot:\n%v", c.help.Usage, help.Usage)
			}
			if help.Description != c.help.Description {
				t.Errorf("expected description:\n%v\ngot:\n%v", c.help.Description, help.Description)
			}
			if !reflect.DeepEqual(c.help.Flags, help.Flags) {
				t.Errorf("expected flags:\n%v\ngot:\n%v", c.help.Flags, help.Flags)
			}
		})
	}
}
