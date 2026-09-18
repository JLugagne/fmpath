package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func run(opts *options, w io.Writer) error {
	if len(opts.sets) > 0 {
		for _, file := range opts.files {
			if err := applySets(file, opts.sets); err != nil {
				return err
			}
		}
	}

	switch {
	case len(opts.gets) > 0:
		return outputGets(w, opts)
	case len(opts.sets) == 0:
		return outputWholeFrontmatter(w, opts.files)
	default:
		return nil
	}
}

// setFrontmatter applies sets to raw file data and returns the new file
// content. It never touches the filesystem.
func setFrontmatter(data []byte, sets []setOp) ([]byte, error) {
	fm, body, has := splitFrontmatter(data)
	root := emptyMapping()
	if has {
		var err error
		root, err = parseFrontmatter(fm)
		if err != nil {
			return nil, err
		}
	}

	for _, op := range sets {
		mapSet(root, op.parts, parseSetValue(op.value))
	}

	fmOut, err := serializeFrontmatter(root)
	if err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(make([]byte, 0, len(fmOut)+len(body)+8))
	buf.WriteString(delimiter)
	buf.WriteByte('\n')
	buf.Write(fmOut)
	buf.WriteString(delimiter)
	buf.WriteByte('\n')
	buf.Write(body)
	return buf.Bytes(), nil
}

func applySets(file string, sets []setOp) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	out, err := setFrontmatter(data, sets)
	if err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}

	perm := os.FileMode(0o644)
	if info, statErr := os.Stat(file); statErr == nil {
		perm = info.Mode().Perm()
	}
	return os.WriteFile(file, out, perm)
}

func outputGets(w io.Writer, opts *options) error {
	root := emptyMapping()

	for _, file := range opts.files {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		fm, _, has := splitFrontmatter(data)
		if !has {
			continue
		}
		src, err := parseFrontmatter(fm)
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}

		var fileOut *yaml.Node
		for _, g := range opts.gets {
			val, ok := mapGet(src, g.parts)
			if !ok {
				continue
			}
			if fileOut == nil {
				fileOut = emptyMapping()
				setKey(root, file, fileOut)
			}
			mapSet(fileOut, g.parts, val)
		}
	}

	out, err := encodeNode(root)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

// outputWholeFrontmatter prints the raw frontmatter of each file, indented,
// under its file name. It avoids parsing and re-encoding the YAML entirely,
// which keeps the common read case cheap and formatting-preserving.
func outputWholeFrontmatter(w io.Writer, files []string) error {
	var buf bytes.Buffer
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}

		fm, _, has := splitFrontmatter(data)
		if !has {
			continue
		}

		key := encodeScalar(file)
		buf.WriteString(key)
		buf.WriteString(":\n")
		indentBlock(&buf, fm)
	}
	_, err := w.Write(buf.Bytes())
	return err
}

func encodeScalar(s string) string {
	out, _ := yaml.Marshal(s)
	return strings.TrimRight(string(out), "\n")
}

func indentBlock(buf *bytes.Buffer, data []byte) {
	for len(data) > 0 {
		idx := bytes.IndexByte(data, '\n')
		var line []byte
		if idx < 0 {
			line, data = data, nil
		} else {
			line, data = data[:idx], data[idx+1:]
		}
		if len(line) > 0 {
			buf.WriteString("  ")
			buf.Write(line)
		}
		buf.WriteByte('\n')
	}
}
