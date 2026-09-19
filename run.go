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
		if opts.oneLine {
			return outputWholeFrontmatterOneLine(w, opts.files)
		}
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

	var out []byte
	var err error
	if opts.oneLine {
		out = encodeOneLine(root)
	} else {
		out, err = encodeNode(root)
		if err != nil {
			return err
		}
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

// outputWholeFrontmatterOneLine is like outputWholeFrontmatter but renders each
// file's top-level keys on a single line: "file.md: key: value key: value".
func outputWholeFrontmatterOneLine(w io.Writer, files []string) error {
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
		root, err := parseFrontmatter(fm)
		if err != nil {
			return fmt.Errorf("%s: %w", file, err)
		}

		buf.WriteString(encodeScalar(file))
		buf.WriteByte(':')
		if pairs := inlinePairs(root); pairs != "" {
			buf.WriteByte(' ')
			buf.WriteString(pairs)
		}
		buf.WriteByte('\n')
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// encodeOneLine renders a mapping of file name to mapping on one line per file.
func encodeOneLine(root *yaml.Node) []byte {
	var buf bytes.Buffer
	for i := 0; i+1 < len(root.Content); i += 2 {
		buf.WriteString(encodeScalar(root.Content[i].Value))
		buf.WriteByte(':')
		if pairs := inlinePairs(root.Content[i+1]); pairs != "" {
			buf.WriteByte(' ')
			buf.WriteString(pairs)
		}
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

// inlinePairs renders the key/value pairs of a mapping joined by spaces without
// surrounding braces; inlinePairsSep allows a custom separator.
func inlinePairs(m *yaml.Node) string {
	return inlinePairsSep(m, " ")
}

func inlinePairsSep(m *yaml.Node, sep string) string {
	if m == nil || m.Kind != yaml.MappingNode {
		return ""
	}
	parts := make([]string, 0, len(m.Content)/2)
	for i := 0; i+1 < len(m.Content); i += 2 {
		parts = append(parts, inlineNode(m.Content[i])+": "+inlineNode(m.Content[i+1]))
	}
	return strings.Join(parts, sep)
}

// inlineNode renders a YAML node on a single line, using flow-style braces and
// brackets for nested collections.
func inlineNode(n *yaml.Node) string {
	if n == nil {
		return ""
	}
	switch n.Kind {
	case yaml.ScalarNode:
		return inlineScalar(n)
	case yaml.SequenceNode:
		parts := make([]string, len(n.Content))
		for i, c := range n.Content {
			parts[i] = inlineNode(c)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case yaml.MappingNode:
		return "{" + inlinePairsSep(n, ", ") + "}"
	case yaml.AliasNode:
		if n.Alias != nil {
			return inlineNode(n.Alias)
		}
	}
	return n.Value
}

func inlineScalar(n *yaml.Node) string {
	if n.Style == yaml.LiteralStyle || n.Style == yaml.FoldedStyle {
		return strings.Join(strings.Fields(n.Value), " ")
	}
	return n.Value
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
