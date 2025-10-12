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

	"github.com/muesli/mango"
	"github.com/muesli/roff"
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
	Name        string
	Usage       string
	Description string
	Flags       []*Flag

	scanner *bufio.Scanner
}

func NewHelp(name string, help io.Reader) *Help {
	return &Help{
		Name:    name,
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
	h.Description = description.String()
	return h.scanner.Err()
}

func (h *Help) toCommand() *mango.Command {
	cmd := mango.NewCommand(h.Name, "", h.Usage)
	for _, f := range h.Flags {
		name := f.Name
		if f.Arg != "" {
			name += " " + f.Arg
		}
		err := cmd.AddFlag(mango.Flag{
			Name:  name,
			Usage: f.Usage,
		})
		if err != nil {
			panic(err)
		}
	}
	return cmd
}

// BuilderWrapper is a wrapper of [mango.Builder] that allows to customize its
// output. Here are the list of its features:
//   - respect SOURCE_DATE_EPOCH environment variable.
//   - customise the manual name
type BuilderWrapper struct {
	mango.Builder
}

func (b BuilderWrapper) Heading(section uint, title, description string, ts time.Time) {
	if epoch := os.Getenv("SOURCE_DATE_EPOCH"); epoch != "" {
		unixEpoch, err := strconv.ParseInt(epoch, 10, 64)
		if err != nil {
			panic("invalid SOURCE_DATE_EPOCH: " + err.Error())
		}
		ts = time.Unix(unixEpoch, 0)
	}
	switch section {
	case 8:
		description = "System Administration Utilities"
	case 6:
		description = "Games"
	default:
		description = "User Commands"
	}
	b.Builder.Heading(section, title, description, ts)
}

type Section struct {
	Title string
	Text  string
}

type Include struct {
	Name        string
	Description string
	Sections    []*Section
}

func parseInclude(path string) (*Include, error) {
	i := &Include{}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	var s *Section
	var text strings.Builder
	finaliseSection := func() error {
		if s != nil {
			s.Text = strings.TrimSpace(text.String())
			if s.Title == "NAME" {
				n, d, ok := strings.Cut(s.Text, " - ")
				if !ok {
					return fmt.Errorf("invalid [name] section")
				}
				i.Name, i.Description = n, d
			} else {
				i.Sections = append(i.Sections, s)
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

	var include *Include
	var err error
	if flagInclude != "" {
		include, err = parseInclude(flagInclude)
		if err != nil {
			l.Fatalln("parse include:", err)
		}
	}

	out, err := getHelp(exe)
	if err != nil {
		l.Fatalln("get help:", err)
	}
	help := NewHelp(filepath.Base(exe), bytes.NewBuffer(out))
	err = help.parse()
	if err != nil {
		l.Fatalln("parse output:", err)
	}
	cmd := help.toCommand()

	description := "manual page for " + cmd.Name
	if include != nil && include.Name != "" {
		cmd.Name = include.Name
		description = include.Description
	}
	if flagName != "" {
		description = flagName
	}
	page := mango.NewManPage(flagSection, cmd.Name, description)
	page.WithLongDescription(help.Description)
	page.Root = *cmd
	if include != nil {
		for _, s := range include.Sections {
			page.WithSection(s.Title, s.Text)
		}
	}
	fmt.Println(page.Build(BuilderWrapper{roff.NewDocument()}))
}
