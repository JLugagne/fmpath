package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	os.Exit(mainRun(os.Args[1:], os.Stdout, os.Stderr))
}

func mainRun(args []string, stdout, stderr io.Writer) int {
	opts, err := parseArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, "fmpath:", err)
		return 2
	}
	if err := run(opts, stdout); err != nil {
		fmt.Fprintln(stderr, "fmpath:", err)
		return 1
	}
	return 0
}
