package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSplitFrontmatter(t *testing.T) {
	data := []byte("---\nid: 1\n---\nbody\n")
	fm, body, has := splitFrontmatter(data)
	if !has {
		t.Fatal("expected frontmatter")
	}
	if string(fm) != "id: 1\n" {
		t.Errorf("fm = %q", fm)
	}
	if string(body) != "body\n" {
		t.Errorf("body = %q", body)
	}

	fm, body, has = splitFrontmatter([]byte("# hello\n"))
	if has || fm != nil || string(body) != "# hello\n" {
		t.Errorf("no-frontmatter case: fm=%q body=%q has=%v", fm, body, has)
	}
}

func TestParseSetValue(t *testing.T) {
	tests := []struct {
		in  string
		tag string
		val string
	}{
		{"hello", "!!str", "hello"},
		{"42", "!!int", "42"},
		{"true", "!!bool", "true"},
		{"", "!!str", ""},
		{"a: b", "!!str", "a: b"},
	}
	for _, tt := range tests {
		n := parseSetValue(tt.in)
		if n.Tag != tt.tag || n.Value != tt.val {
			t.Errorf("parseSetValue(%q) = {%s %q}, want {%s %q}", tt.in, n.Tag, n.Value, tt.tag, tt.val)
		}
	}
}

func TestMapSetGet(t *testing.T) {
	root := emptyMapping()
	mapSet(root, []string{"a", "b", "c"}, parseSetValue("1"))
	mapSet(root, []string{"a", "d"}, parseSetValue("x"))

	got, ok := mapGet(root, []string{"a", "b", "c"})
	if !ok || got.Value != "1" {
		t.Fatalf("get a.b.c = %v, %v", got, ok)
	}
	got, ok = mapGet(root, []string{"a", "d"})
	if !ok || got.Value != "x" {
		t.Fatalf("get a.d = %v, %v", got, ok)
	}
	if _, ok := mapGet(root, []string{"a", "z"}); ok {
		t.Error("unexpected match for a.z")
	}

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("a:\n  b: 1\n"), &doc); err != nil {
		t.Fatal(err)
	}
}

func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunGet(t *testing.T) {
	f1 := writeTemp(t, "f1.md", "---\nid: XX\nname: Alice\n---\nbody\n")
	f2 := writeTemp(t, "f2.md", "---\nid: YY\nname: Bob\n---\nbody\n")

	opts, err := parseArgs([]string{"--get", "id", "--get", "name", f1, f2})
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := run(opts, &buf); err != nil {
		t.Fatal(err)
	}

	want := f1 + ":\n  id: XX\n  name: Alice\n" + f2 + ":\n  id: YY\n  name: Bob\n"
	if buf.String() != want {
		t.Errorf("output =\n%s\nwant\n%s", buf.String(), want)
	}
}

func TestRunGetNestedAndMissing(t *testing.T) {
	f1 := writeTemp(t, "f1.md", "---\nmeta:\n  level: 3\n---\nbody\n")

	opts, _ := parseArgs([]string{"--get", ".meta.level", "--get", "nope", f1})
	var buf bytes.Buffer
	if err := run(opts, &buf); err != nil {
		t.Fatal(err)
	}
	want := f1 + ":\n  meta:\n    level: 3\n"
	if buf.String() != want {
		t.Errorf("output =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestRunSet(t *testing.T) {
	f1 := writeTemp(t, "f1.md", "---\nid: 1\nnested:\n  keep: yes\n---\n# Title\ntext\n")

	opts, err := parseArgs([]string{"--set", ".id=2", "--set", ".nested.new=deep", f1})
	if err != nil {
		t.Fatal(err)
	}
	if err := run(opts, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(f1)
	if err != nil {
		t.Fatal(err)
	}
	want := "---\nid: 2\nnested:\n  keep: yes\n  new: deep\n---\n# Title\ntext\n"
	if string(got) != want {
		t.Errorf("file =\n%s\nwant\n%s", got, want)
	}
}

func TestRunSetCreatesFrontmatter(t *testing.T) {
	f1 := writeTemp(t, "f1.md", "# No frontmatter\n")

	opts, _ := parseArgs([]string{"--set", ".title=Hello", f1})
	if err := run(opts, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(f1)
	want := "---\ntitle: Hello\n---\n# No frontmatter\n"
	if string(got) != want {
		t.Errorf("file =\n%q\nwant\n%q", got, want)
	}
}
