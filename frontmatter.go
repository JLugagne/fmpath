package main

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

const delimiter = "---"

func emptyMapping() *yaml.Node {
	return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
}

func scalarNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

// splitFrontmatter separates the YAML frontmatter from the markdown body.
// It returns has=false when the file does not start with a frontmatter block.
func splitFrontmatter(data []byte) (fm []byte, body []byte, has bool) {
	if !firstLineIsDelimiter(data) {
		return nil, data, false
	}

	start := lineEnd(data, 0)
	i := start
	for i < len(data) {
		le := lineEnd(data, i)
		line := bytes.TrimRight(data[i:le], "\r\n")
		if bytes.Equal(line, []byte(delimiter)) || bytes.Equal(line, []byte("...")) {
			return data[start:i], data[le:], true
		}
		i = le
	}

	return nil, data, false
}

func firstLineIsDelimiter(data []byte) bool {
	le := lineEnd(data, 0)
	line := bytes.TrimRight(data[:le], "\r\n")
	return bytes.Equal(line, []byte(delimiter))
}

// lineEnd returns the index just after the line starting at start.
func lineEnd(data []byte, start int) int {
	if start >= len(data) {
		return len(data)
	}
	if idx := bytes.IndexByte(data[start:], '\n'); idx >= 0 {
		return start + idx + 1
	}
	return len(data)
}

func parseFrontmatter(fm []byte) (*yaml.Node, error) {
	if len(bytes.TrimSpace(fm)) == 0 {
		return emptyMapping(), nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(fm, &doc); err != nil {
		return nil, err
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return emptyMapping(), nil
	}

	root := doc.Content[0]
	switch root.Kind {
	case yaml.MappingNode:
		return root, nil
	case yaml.ScalarNode:
		if root.Tag == "!!null" {
			return emptyMapping(), nil
		}
	}
	return nil, fmt.Errorf("frontmatter is not a YAML mapping")
}

func serializeFrontmatter(root *yaml.Node) ([]byte, error) {
	return encodeNode(root)
}

func encodeNode(root *yaml.Node) ([]byte, error) {
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// mapGet walks a mapping node following parts and returns the matched node.
func mapGet(m *yaml.Node, parts []string) (*yaml.Node, bool) {
	cur := m
	for _, p := range parts {
		if cur == nil || cur.Kind != yaml.MappingNode {
			return nil, false
		}
		next := getKey(cur, p)
		if next == nil {
			return nil, false
		}
		cur = next
	}
	return cur, true
}

func getKey(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// mapSet writes a deep copy of value at the given path, creating intermediate
// mappings when needed.
func mapSet(m *yaml.Node, parts []string, value *yaml.Node) {
	cur := m
	for i, p := range parts {
		if i == len(parts)-1 {
			setKey(cur, p, value)
			return
		}

		child := getKey(cur, p)
		if child == nil || child.Kind != yaml.MappingNode {
			child = emptyMapping()
			setKey(cur, p, child)
		}
		cur = child
	}
}

func setKey(m *yaml.Node, key string, value *yaml.Node) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1] = value
			return
		}
	}
	m.Content = append(m.Content, scalarNode(key), value)
}

// parseSetValue interprets a raw string as a YAML scalar when possible so that
// numbers, booleans and null keep their type. Complex values fall back to a
// plain string.
func parseSetValue(raw string) *yaml.Node {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(raw), &doc); err == nil &&
		doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		if n := doc.Content[0]; n.Kind == yaml.ScalarNode {
			return n
		}
	}
	return scalarNode(raw)
}
