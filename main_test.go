package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseUsage(t *testing.T) {
	cases := []struct {
		name  string
		val   string
		usage string
	}{
		{
			"empty string",
			"",
			"",
		},
		{
			"short flag",
			"  -h	Show help and exit.",
			"",
		},
		{
			"go flag default",
			"Usage of gohelp2man:",
			"",
		},
		{
			"custom GNU-like",
			"Usage: gohelp2man [OPTION]... EXECUTABLE",
			"gohelp2man [OPTION]... EXECUTABLE",
		},
		{
			"multiline GNU-like",
			`Usage: ln [OPTION]... [-T] TARGET LINK_NAME
  or:  ln [OPTION]... TARGET
  or:  ln [OPTION]... TARGET... DIRECTORY
  or:  ln [OPTION]... -t DIRECTORY TARGET...
In the 1st form, create a link to TARGET with the name LINK_NAME.`,
			`ln [OPTION]... [-T] TARGET LINK_NAME
ln [OPTION]... TARGET
ln [OPTION]... TARGET... DIRECTORY
ln [OPTION]... -t DIRECTORY TARGET...`,
		},
		{
			"multiline go-like",
			`Usage of stringer:
	stringer [flags] -type T [directory]
	stringer [flags] -type T files... # Must be a single package
For more information, see:
	https://pkg.go.dev/golang.org/x/tools/cmd/stringer`,
			`stringer [flags] -type T [directory]
stringer [flags] -type T files... # Must be a single package`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			help := NewHelp(strings.NewReader(c.val))
			help.scanner.Scan()
			help.parseUsage()
			if !reflect.DeepEqual(c.usage, help.Usage) {
				t.Fatalf("expected:\n%v\ngot:\n%v", c.usage, help.Usage)
			}
		})
	}
}

func TestParseFlags(t *testing.T) {
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
			"multi short",
			`  -h	Show help
    	and exit.`,
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
			"multi long",
			`  -help
    	Show help
    	and exit.`,
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
			help.parseFlags()
			if c.found {
				if len(help.Flags) == 0 {
					t.Fatal("expected to get one flag, got none")
				}
				f := help.Flags[0]
				if !reflect.DeepEqual(c.flag, f) {
					t.Fatalf("expected:\n%v\ngot:\n%v", c.flag, f)
				}
			} else {
				if len(help.Flags) != 0 {
					t.Fatal("expected to get no flags, got ", help.Flags)
				}
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

func TestWriteSynopsis(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{"basic", "test [OPTION]... [ARGUMENT]...", `\fBtest\fR [\fIOPTION\fR]... [\fIARGUMENT\fR]...`},
		{"non-closed brackets", "test [argument", `\fBtest\fR [argument`},
		{"end with lbracket", "test argument[", `\fBtest\fR argument[`},
		{"end with rbracket", "test argument]", `\fBtest\fR argument]`},
		{"single bracketed arg", "test [argument]", `\fBtest\fR [\fIargument\fR]`},
		{"no args", "test", `\fBtest\fR`},
		{"no args with space", "test ", `\fBtest\fR`},
		{"empty", "", `\fB\fR`},
		{"single space", "", `\fB\fR`},
		{"starts with space", " test args", `\fBtest\fR args`},
		{
			"basic multiline",
			`stringer [flags] -type T [directory]
stringer [flags] -type T files...`,
			`\fBstringer\fR [\fIflags\fR] \-type T [\fIdirectory\fR]
.br
\fBstringer\fR [\fIflags\fR] \-type T files...`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := &strings.Builder{}
			writeSynopsis(w, c.input)
			actual := strings.TrimSuffix(w.String(), "\n")
			if actual != c.expected {
				t.Fatalf("expected:\n%s\ngot:\n%s", c.expected, actual)
			}
		})
	}
}

func setup(t *testing.T) string {
	t.Helper()
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	tmp := t.TempDir()
	out := filepath.Join(tmp, "out")
	os.Args = []string{"gohelp2man", "-output", out, "testdata/test.sh"}
	return out
}

func TestFull(t *testing.T) {
	cases := []string{
		"basic",
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			out := setup(t)
			t.Setenv("GOHELP2MAN_TESTCASE", filepath.Join("testdata", "test_full_"+c+".txt"))
			t.Setenv("SOURCE_DATE_EPOCH", "0")
			main()
			expected, err := os.ReadFile(filepath.Join("testdata", "test_full_"+c+".1"))
			if err != nil {
				t.Fatal(err)
			}
			actual, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(expected, actual) {
				t.Errorf("expected:\n%s\ngot:\n%s", expected, actual)
			}
		})
	}
}
