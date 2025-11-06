package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
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
			&Flag{"h", "", "Show help\nand exit."},
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
			&Flag{"help", "", "Show help\nand exit."},
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
				Usage: "test [OPTION]... ARG",
				Sections: map[string]*Section{
					"DESCRIPTION": {"DESCRIPTION", "A test help message.", 0},
				},
			},
		},
		{
			name: "description after usage",
			val: `Usage: test [OPTION]... ARG

A test help message.
`,
			help: &Help{
				Usage: "test [OPTION]... ARG",
				Sections: map[string]*Section{
					"DESCRIPTION": {"DESCRIPTION", "A test help message.", 0},
				},
			},
		},
		{
			name: "description after flags",
			val: `Usage: test [OPTION]... ARG
  -h	Show help.

A test help message.
`,
			help: &Help{
				Usage: "test [OPTION]... ARG",
				Flags: []*Flag{{"h", "", "Show help."}},
				Sections: map[string]*Section{
					"DESCRIPTION": {"DESCRIPTION", "A test help message.", 0},
				},
			},
		},
		{
			name: "options header",
			val: `Text of the description.

Options:
  -h	Show help.
`,
			help: &Help{
				Flags: []*Flag{{"h", "", "Show help."}},
				Sections: map[string]*Section{
					"DESCRIPTION": {"DESCRIPTION", "Text of the description.", 0},
				},
			},
		},
		{
			name: "unknown section header",
			val: `Other section:
Text of this section.
`,
			help: &Help{Sections: map[string]*Section{
				"DESCRIPTION": {"DESCRIPTION", ".SS Other section:\nText of this section.", 0},
			}},
		},
		{
			name: "known header after flags",
			val: `Text of the description.
  -h	Show help.
Author:
Nicolas Peugnet
`,
			help: &Help{
				Flags: []*Flag{{"h", "", "Show help."}},
				Sections: map[string]*Section{
					"DESCRIPTION": {"DESCRIPTION", "Text of the description.", 0},
					"AUTHOR":      {"AUTHOR", "Nicolas Peugnet", 0},
				},
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
			if !reflect.DeepEqual(c.help.Flags, help.Flags) {
				t.Errorf("expected flags:\n%v\ngot:\n%v", c.help.Flags, help.Flags)
			}
			if c.help.Sections == nil {
				c.help.Sections = make(map[string]*Section)
			}
			if !reflect.DeepEqual(c.help.Sections, help.Sections) {
				t.Errorf("expected sections:\n%v\ngot:\n%v", c.help.Sections, help.Sections)
			}
		})
	}
}

func TestParseInclude(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected *Include
	}{
		{
			"empty",
			"",
			&Include{Sections: map[string]*Section{}},
		},
		{
			"single known section",
			`[NAME]
gohelp2man - generate a simple manual page for Go programs
`,
			&Include{Sections: map[string]*Section{
				"NAME": {
					Title: "NAME",
					Text:  "gohelp2man - generate a simple manual page for Go programs",
				},
			}},
		},
		{
			"lowercase known section",
			`[name]
gohelp2man - generate a simple manual page for Go programs
`,
			&Include{Sections: map[string]*Section{
				"NAME": {
					Title: "NAME",
					Text:  "gohelp2man - generate a simple manual page for Go programs",
				},
			}},
		},
		{
			"single other section",
			`[Other section]
This is a section that is not known.
`,
			&Include{Sections: map[string]*Section{}, OtherSections: []*Section{
				{
					Title: "OTHER SECTION",
					Text:  "This is a section that is not known.",
				},
			}},
		},
		{
			"positionned known section",
			"[>DESCRIPTION]\nAppend\n",
			&Include{Sections: map[string]*Section{
				"DESCRIPTION": {"DESCRIPTION", "Append", '>'},
			}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual, err := parseInclude(strings.NewReader(c.input))
			if err != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(c.expected, actual) {
				t.Fatalf("expected %v, got %v", c.expected, actual)
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
		"escapes",
		"with_headers",
	}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			basename := filepath.Join("testdata", "test_full_"+c)
			out := setup(t)
			last := len(os.Args) - 1
			os.Args = append(os.Args[:last], "-opt-include", basename+".h2m", os.Args[last])
			t.Setenv("GOHELP2MAN_TESTCASE", basename+".txt")
			t.Setenv("SOURCE_DATE_EPOCH", "0")
			main()
			expected, err := os.ReadFile(basename + ".1")
			if err != nil {
				t.Fatal(err)
			}
			actual, err := os.ReadFile(out)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(expected, actual) {
				cmd := exec.Command("diff", "-u", "--label=expected", "--label=got", basename+".1", out)
				diff, err := cmd.Output()
				exitErr := &exec.ExitError{}
				if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
					t.Errorf("\n%s", diff)
				} else {
					t.Errorf("expected:\n%s\ngot:\n%s", expected, actual)
				}
			}
		})
	}
}
