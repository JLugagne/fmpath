package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(mainRun(os.Args[1:], os.Stdout, os.Stderr))
}

const helpText = `fmpath - read and write YAML frontmatter of Markdown files

Usage:
  fmpath [--get PATH]... [--set PATH=VALUE]... [--one-line] FILE...
  fmpath --help

Read fields:
  fmpath --get id --get .meta.author.name *.md

Write fields:
  fmpath --set .id=42 --set .draft=true post.md

Print the whole frontmatter (no --get):
  fmpath *.md

Print one line per file:
  fmpath --one-line *.md

Flags:
  --get PATH        Select a field to read; repeatable. The leading dot is optional.
  --set PATH=VALUE  Set a field; repeatable. Values are parsed as YAML scalars when possible.
  --one-line        Render each file's keys on a single line. -one-line works too.
  --help, -h        Show this help and exit.
  --                Treat every following argument as a file.

Exit codes:
  0  Success
  1  Runtime error (e.g. unreadable file, invalid frontmatter)
  2  Usage error (bad flags or no file given)
`

func mainRun(args []string, stdout, stderr io.Writer) int {
	opts, err := parseArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, "fmpath:", err)
		return 2
	}
	if opts.help {
		fmt.Fprint(stdout, helpText)
		return 0
	}
	if err := run(opts, stdout); err != nil {
		fmt.Fprintln(stderr, "fmpath:", err)
		return 1
	}
	return 0
}
