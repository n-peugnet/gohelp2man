package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const (
	Name  = "gohelp2man"
	Usage = `%s generates a man page out of a Go program's -help output.

Usage: %s [OPTION]... EXECUTABLE
`

	RegexSection = `^\[([^]]+)\]\s*$`
	RegexUsage   = `[Uu]sage(:| of) (?U:(.*)):?$`
	RegexFlag    = `^  -((\w)\t(.*)|([-\w]+) (.+)|[-\w]+)$`
)

var (
	l            = log.New(os.Stderr, Name+": ", 0)
	regexSection = regexp.MustCompile(RegexSection)
	regexUsage   = regexp.MustCompile(RegexUsage)
	regexFlag    = regexp.MustCompile(RegexFlag)
)

type Flag struct {
	Name  string
	Arg   string
	Usage string
}

func (f *Flag) String() string {
	return fmt.Sprintf("-%s %q: %s", f.Name, f.Arg, f.Usage)
}

type Help struct {
	Usage       string
	Description string
	Flags       []*Flag

	scanner *bufio.Scanner
}

func NewHelp(help io.Reader) *Help {
	return &Help{
		scanner: bufio.NewScanner(help),
	}
}

func (h *Help) parseUsage() (usage string, found bool) {
	line := h.scanner.Text()
	m := regexUsage.FindStringSubmatch(line)
	if m != nil {
		return m[2], true
	}
	return "", false
}

func (h *Help) parseFlag() (f *Flag, found bool) {
	line := h.scanner.Text()
	m := regexFlag.FindStringSubmatch(line)
	found = m != nil
	if found {
		f = new(Flag)
		switch {
		case m[2] != "":
			f.Name = m[2]
			f.Usage = m[3]
			return
		case m[4] != "":
			f.Name = m[4]
			f.Arg = m[5]
		default:
			f.Name = m[1]
		}
		if !h.scanner.Scan() {
			panic("missing description for long flag: " + f.Name)
		}
		f.Usage = strings.TrimSpace(h.scanner.Text())
	}
	return
}

func (h *Help) parse() error {
	description := strings.Builder{}
	for h.scanner.Scan() {
		if u, found := h.parseUsage(); found {
			h.Usage = u
			continue
		}
		if f, found := h.parseFlag(); found {
			h.Flags = append(h.Flags, f)
			continue
		}
		description.Write(h.scanner.Bytes())
		description.WriteString("\n")
	}
	h.Description = strings.TrimSpace(description.String())
	return h.scanner.Err()
}

func now() time.Time {
	if epoch := os.Getenv("SOURCE_DATE_EPOCH"); epoch != "" {
		unixEpoch, err := strconv.ParseInt(epoch, 10, 64)
		if err != nil {
			panic("invalid SOURCE_DATE_EPOCH: " + err.Error())
		}
		return time.Unix(unixEpoch, 0)
	} else {
		return time.Now()
	}
}

var KnownSections = [12]string{
	"NAME",
	"SYNOPSIS",
	"DESCRIPTION",
	"OPTIONS",
	// Other
	"ENVIRONMENT",
	"FILES",
	"EXAMPLES",
	"AUTHOR",
	"REPORTING BUGS",
	"COPYRIGHT",
	"SEE ALSO",
}

type Section struct {
	Title string
	Text  string
}

type Include struct {
	Name          string
	Description   string
	Sections      map[string]*Section
	OtherSections []*Section
}

func parseInclude(path string) (*Include, error) {
	i := &Include{Sections: make(map[string]*Section)}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var s *Section
	var text strings.Builder
	finaliseSection := func() error {
		if s != nil {
			s.Text = strings.TrimSpace(text.String())
			switch s.Title {
			case "NAME",
				"SYNOPSIS",
				"DESCRIPTION",
				"OPTIONS",
				"ENVIRONMENT",
				"FILES",
				"EXAMPLES",
				"AUTHOR",
				"REPORTING BUGS",
				"COPYRIGHT",
				"SEE ALSO":
				i.Sections[s.Title] = s
			default:
				i.OtherSections = append(i.OtherSections, s)
			}
		}
		text.Reset()
		return nil
	}

	scanner := bufio.NewScanner(bufio.NewReader(file))
	for scanner.Scan() {
		line := scanner.Text()
		m := regexSection.FindStringSubmatch(line)
		if m != nil {
			if err := finaliseSection(); err != nil {
				return nil, err
			}
			s = &Section{Title: strings.ToUpper(m[1])}
			continue
		}
		text.WriteString(line)
		text.WriteString("\n")
	}
	if err := finaliseSection(); err != nil {
		return nil, err
	}
	return i, scanner.Err()
}

func getHelp(exe string) ([]byte, error) {
	cmd := exec.Command(exe, "-help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("run %s: %w", cmd, err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("run %s: empty output", cmd)
	}
	return out, err
}

// writeSynopsis formats a synopsis line by writing the command name in bold
// and the arguments inside brackets in italic.
func writeSynopsis(w io.Writer, synopsis string) {
	name, args, found := strings.Cut(strings.TrimSpace(synopsis), " ")
	fmt.Fprintf(w, "\\fB%s\\fR", name)
	if found {
		fmt.Fprint(w, " ")
	}
	for {
		lBracket := strings.Index(args, "[")
		if lBracket == -1 {
			fmt.Fprint(w, args)
			break
		}
		fmt.Fprint(w, args[:lBracket])
		args = args[lBracket:]
		rBracket := strings.Index(args, "]")
		if rBracket == -1 {
			fmt.Fprint(w, args)
			break
		}
		fmt.Fprint(w, "[")
		fmt.Fprintf(w, "\\fI%s\\fR", args[1:rBracket])
		fmt.Fprint(w, "]")
		args = args[rBracket+1:]
	}
	fmt.Fprintln(w)
}

func main() {
	cli := flag.NewFlagSet(Name, flag.ExitOnError)
	cli.Usage = func() {
		fmt.Fprintf(cli.Output(), Usage, Name, Name)
		cli.PrintDefaults()
	}
	var (
		flagHelp    bool
		flagInclude string
		flagName    string
		flagSection uint
		flagVersion bool
	)
	cli.BoolVar(&flagHelp, "help", false, "Show this help and exit.")
	cli.StringVar(&flagInclude, "include", "", "Include material from `FILE`.")
	cli.StringVar(&flagName, "name", "", "description for the NAME paragraph.")
	cli.UintVar(&flagSection, "section", 1, "section number for manual page (1, 6, 8).")
	cli.BoolVar(&flagVersion, "version", false, "Show version number and exit.")
	cli.Parse(os.Args[1:])

	if flagHelp {
		cli.Usage()
		os.Exit(0)
	}

	if flagVersion {
		v := "(unknown)"
		info, ok := debug.ReadBuildInfo()
		if ok {
			v = info.Main.Version
		}
		fmt.Println(Name, v)
		os.Exit(0)
	}

	exe := cli.Arg(0)
	if exe == "" {
		l.Print("missing argument: executable")
		cli.Usage()
		os.Exit(2)
	}

	include := &Include{}
	if flagInclude != "" {
		var err error
		include, err = parseInclude(flagInclude)
		if err != nil {
			l.Fatalln("parse include:", err)
		}
	}

	out, err := getHelp(exe)
	if err != nil {
		l.Fatalln("get help:", err)
	}
	help := NewHelp(bytes.NewBuffer(out))
	err = help.parse()
	if err != nil {
		l.Fatalln("parse output:", err)
	}

	name := filepath.Base(exe)
	description := "manual page for " + name
	if s, found := include.Sections["NAME"]; found {
		n, d, ok := strings.Cut(s.Text, " - ")
		if !ok {
			l.Fatalf("invalid [name] section %q", s.Text)
		}
		if i := strings.IndexAny(n, " \t\n\r"); i != -1 {
			l.Fatalf("illegal character %q in program name: %q", n[i], n)
		}
		name, description = n, d
	}
	if flagName != "" {
		description = flagName
	}

	b := bufio.NewWriter(os.Stdout)

	// Write title
	fmt.Fprintf(b, ".TH %s %v %q %q\n",
		strings.ToUpper(name), flagSection, now().Format("2006-01-02"), name,
	)

	// Write NAME section
	fmt.Fprintf(b, ".SH NAME\n%v \\- %v\n", name, description)

	// Write SYNOPSIS section
	fmt.Fprintln(b, ".SH SYNOPSIS")
	if s, found := include.Sections["SYNOPSIS"]; found {
		fmt.Fprintln(b, s.Text)
	} else if help.Usage != "" {
		writeSynopsis(b, help.Usage)
	} else {
		fmt.Fprintf(b, "\\fB%s\\fR [\\fIOPTION\\fR]... [\\fIARGUMENT\\fR]...\n", name)
	}

	// Write DESCRIPTION section
	if s, found := include.Sections["DESCRIPTION"]; found {
		fmt.Fprintln(b, s.Text)
	}
	if help.Description != "" {
		fmt.Fprintf(b, ".SH DESCRIPTION\n%s\n", help.Description)
	}

	// Write OPTIONS section
	fmt.Fprint(b, ".SH OPTIONS\n")
	if s, found := include.Sections["OPTIONS"]; found {
		fmt.Fprintln(b, s.Text)
	}
	for _, f := range help.Flags {
		if f.Arg != "" {
			fmt.Fprintf(b, ".TP\n\\fB\\-%s\\fR %s\n", f.Name, f.Arg)
		} else {
			fmt.Fprintf(b, ".TP\n\\fB\\-%s\\fR\n", f.Name)
		}
		fmt.Fprintln(b, f.Usage)
	}

	// Write other included sections
	for _, s := range include.OtherSections {
		fmt.Fprintf(b, ".SH %s\n%s\n", s.Title, s.Text)
	}

	// Write last known sections
	for _, title := range KnownSections[4:] {
		if s, found := include.Sections[title]; found {
			fmt.Fprintf(b, ".SH %s\n%s\n", s.Title, s.Text)
		}
	}

	// Print man page
	b.Flush()
}
