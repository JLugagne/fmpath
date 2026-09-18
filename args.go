package main

import (
	"fmt"
	"strings"
)

type getOp struct {
	raw   string
	parts []string
}

type setOp struct {
	raw   string
	parts []string
	value string
}

type options struct {
	gets  []getOp
	sets  []setOp
	files []string
	help  bool
}

func parseArgs(args []string) (*options, error) {
	opts := &options{}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "--help" || arg == "-h" || arg == "-help":
			opts.help = true
			return opts, nil

		case arg == "--get" || arg == "-get":
			val, next, err := flagValue(args, i, arg)
			if err != nil {
				return nil, err
			}
			parts, err := parsePath(val)
			if err != nil {
				return nil, err
			}
			opts.gets = append(opts.gets, getOp{raw: val, parts: parts})
			i = next

		case strings.HasPrefix(arg, "--get=") || strings.HasPrefix(arg, "-get="):
			val := arg[strings.Index(arg, "=")+1:]
			parts, err := parsePath(val)
			if err != nil {
				return nil, err
			}
			opts.gets = append(opts.gets, getOp{raw: val, parts: parts})

		case arg == "--set" || arg == "-set":
			val, next, err := flagValue(args, i, arg)
			if err != nil {
				return nil, err
			}
			op, err := parseSet(val)
			if err != nil {
				return nil, err
			}
			opts.sets = append(opts.sets, op)
			i = next

		case strings.HasPrefix(arg, "--set=") || strings.HasPrefix(arg, "-set="):
			val := arg[strings.Index(arg, "=")+1:]
			op, err := parseSet(val)
			if err != nil {
				return nil, err
			}
			opts.sets = append(opts.sets, op)

		case arg == "--":
			opts.files = append(opts.files, args[i+1:]...)
			i = len(args)

		case strings.HasPrefix(arg, "-") && arg != "-":
			return nil, fmt.Errorf("unknown flag %q", arg)

		default:
			opts.files = append(opts.files, arg)
		}
	}

	if len(opts.files) == 0 {
		return nil, fmt.Errorf("no markdown file given")
	}

	return opts, nil
}

func flagValue(args []string, i int, name string) (string, int, error) {
	if i+1 >= len(args) {
		return "", i, fmt.Errorf("flag %s needs a value", name)
	}
	return args[i+1], i + 1, nil
}

func parseSet(arg string) (setOp, error) {
	eq := strings.Index(arg, "=")
	if eq < 0 {
		return setOp{}, fmt.Errorf("invalid --set %q, expected .path=value", arg)
	}
	rawPath := arg[:eq]
	value := arg[eq+1:]
	parts, err := parsePath(rawPath)
	if err != nil {
		return setOp{}, err
	}
	return setOp{raw: arg, parts: parts, value: value}, nil
}

func parsePath(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, ".")
	if s == "" {
		return nil, fmt.Errorf("invalid empty path")
	}
	parts := strings.Split(s, ".")
	for _, p := range parts {
		if p == "" {
			return nil, fmt.Errorf("invalid path %q", s)
		}
	}
	return parts, nil
}
