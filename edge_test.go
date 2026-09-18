package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseArgsFlagForms(t *testing.T) {
	opts, err := parseArgs([]string{"-get", "id", "-get=name", "-set", ".a=1", "-set=.b=2", "x.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.gets) != 2 || opts.gets[0].raw != "id" || opts.gets[1].raw != "name" {
		t.Fatalf("gets = %+v", opts.gets)
	}
	if len(opts.sets) != 2 || opts.sets[0].parts[0] != "a" || opts.sets[1].parts[0] != "b" {
		t.Fatalf("sets = %+v", opts.sets)
	}
}

func TestParseArgsDoubleDash(t *testing.T) {
	opts, err := parseArgs([]string{"--get", "id", "--", "--weird.md", "ok.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.files) != 2 || opts.files[0] != "--weird.md" || opts.files[1] != "ok.md" {
		t.Fatalf("files = %v", opts.files)
	}
}

func TestParseArgsDashIsFile(t *testing.T) {
	opts, err := parseArgs([]string{"--get", "id", "-"})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.files) != 1 || opts.files[0] != "-" {
		t.Fatalf("files = %v", opts.files)
	}
}

func TestParseArgsMissingFlagValues(t *testing.T) {
	for _, args := range [][]string{{"--get"}, {"--set"}, {"-get"}, {"-set"}} {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("parseArgs(%v) expected error", args)
		}
	}
}

func TestParseArgsBadSetPath(t *testing.T) {
	if _, err := parseArgs([]string{"--set", ".=1", "a.md"}); err == nil {
		t.Error("expected error for empty path in set")
	}
}

func TestParseArgsBadGetPath(t *testing.T) {
	if _, err := parseArgs([]string{"--get", ".a..b", "a.md"}); err == nil {
		t.Error("expected error for malformed get path")
	}
}

func TestParseArgsInlineFlagErrors(t *testing.T) {
	cases := [][]string{
		{"--get=.a..b", "a.md"},
		{"-get=.a..b", "a.md"},
		{"--set=.=1", "a.md"},
		{"-set=.=1", "a.md"},
	}
	for _, args := range cases {
		if _, err := parseArgs(args); err == nil {
			t.Errorf("parseArgs(%v) expected error", args)
		}
	}
}

func TestSplitFrontmatterEdgeCases(t *testing.T) {
	if _, body, has := splitFrontmatter(nil); has || len(body) != 0 {
		t.Errorf("nil input: body=%q has=%v", body, has)
	}

	data := []byte("---\nid: 1\nbody\n")
	if _, _, has := splitFrontmatter(data); has {
		t.Error("unterminated frontmatter should not be detected")
	}

	fm, body, has := splitFrontmatter([]byte("---\nid: 1\n...\nbody\n"))
	if !has || string(fm) != "id: 1\n" || string(body) != "body\n" {
		t.Errorf("... terminator: fm=%q body=%q has=%v", fm, body, has)
	}

	fm, body, has = splitFrontmatter([]byte("---\r\nid: 1\r\n---\r\nbody\r\n"))
	if !has || string(fm) != "id: 1\r\n" || string(body) != "body\r\n" {
		t.Errorf("crlf: fm=%q body=%q has=%v", fm, body, has)
	}

	if _, _, has := splitFrontmatter([]byte("---")); has {
		t.Error("lone delimiter should not be frontmatter")
	}
}

func TestParseFrontmatterErrors(t *testing.T) {
	if root, err := parseFrontmatter(nil); err != nil || root.Kind != yaml.MappingNode {
		t.Errorf("nil fm: root=%v err=%v", root, err)
	}
	if root, err := parseFrontmatter([]byte("   \n")); err != nil || root.Kind != yaml.MappingNode {
		t.Errorf("blank fm: root=%v err=%v", root, err)
	}
	if root, err := parseFrontmatter([]byte("null")); err != nil || root.Kind != yaml.MappingNode {
		t.Errorf("null fm: root=%v err=%v", root, err)
	}
	if root, err := parseFrontmatter([]byte("# only a comment\n")); err != nil || root.Kind != yaml.MappingNode {
		t.Errorf("comment-only fm: root=%v err=%v", root, err)
	}
	if _, err := parseFrontmatter([]byte("a: [1,")); err == nil {
		t.Error("expected yaml error")
	}
	if _, err := parseFrontmatter([]byte("just a string")); err == nil {
		t.Error("expected non-mapping error")
	}
}

func TestMapGetThroughScalar(t *testing.T) {
	root := emptyMapping()
	mapSet(root, []string{"a"}, parseSetValue("1"))
	if _, ok := mapGet(root, []string{"a", "b"}); ok {
		t.Error("expected no match through scalar")
	}
}

func TestParseSetValueFallback(t *testing.T) {
	n := parseSetValue("a: [1,")
	if n.Kind != yaml.ScalarNode || n.Value != "a: [1," {
		t.Errorf("fallback = %v", n)
	}
}

func TestEncodeNodeError(t *testing.T) {
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{scalarNode("x")}}
	if _, err := encodeNode(doc); err == nil {
		t.Error("expected encode error for nested document node")
	}
}

func TestRunErrors(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.md")

	opts, _ := parseArgs([]string{"--get", "id", missing})
	if err := run(opts, &bytes.Buffer{}); err == nil {
		t.Error("expected get on missing file to fail")
	}

	opts, _ = parseArgs([]string{"--set", ".id=1", missing})
	if err := run(opts, &bytes.Buffer{}); err == nil {
		t.Error("expected set on missing file to fail")
	}

	bad := writeTemp(t, "bad.md", "---\na: [1,\n---\nbody\n")
	opts, _ = parseArgs([]string{"--get", "a", bad})
	if err := run(opts, &bytes.Buffer{}); err == nil {
		t.Error("expected get on invalid frontmatter to fail")
	}
	opts, _ = parseArgs([]string{"--set", ".a=1", bad})
	if err := run(opts, &bytes.Buffer{}); err == nil {
		t.Error("expected set on invalid frontmatter to fail")
	}
}

func TestOutputGetsSkipsFilesWithoutFrontmatter(t *testing.T) {
	plain := writeTemp(t, "plain.md", "# no fm\n")
	withFM := writeTemp(t, "fm.md", "---\nid: 1\n---\nbody\n")

	opts, _ := parseArgs([]string{"--get", "id", plain, withFM})
	var buf bytes.Buffer
	if err := run(opts, &buf); err != nil {
		t.Fatal(err)
	}
	want := withFM + ":\n  id: 1\n"
	if buf.String() != want {
		t.Errorf("output = %q, want %q", buf.String(), want)
	}
}

func TestRunWholeFrontmatterWhenNoGet(t *testing.T) {
	withFM := writeTemp(t, "fm.md", "---\nid: XX\nname: Alice\nmeta:\n  level: 3\n---\nbody\n")
	plain := writeTemp(t, "plain.md", "# no fm\n")

	opts, err := parseArgs([]string{withFM, plain})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := run(opts, &buf); err != nil {
		t.Fatal(err)
	}
	want := withFM + ":\n  id: XX\n  name: Alice\n  meta:\n    level: 3\n"
	if buf.String() != want {
		t.Errorf("output = %q, want %q", buf.String(), want)
	}
}

func TestRunWholeFrontmatterReadError(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.md")
	opts, _ := parseArgs([]string{missing})
	if err := run(opts, &bytes.Buffer{}); err == nil {
		t.Error("expected read error for whole frontmatter")
	}
}

func TestIndentBlock(t *testing.T) {
	var buf bytes.Buffer
	indentBlock(&buf, []byte("id: 1\nname: x\n"))
	if buf.String() != "  id: 1\n  name: x\n" {
		t.Errorf("indentBlock = %q", buf.String())
	}

	buf.Reset()
	indentBlock(&buf, []byte("id: 1"))
	if buf.String() != "  id: 1\n" {
		t.Errorf("indentBlock no newline = %q", buf.String())
	}

	buf.Reset()
	indentBlock(&buf, []byte("a:\n\n  b: 1\n"))
	if buf.String() != "  a:\n\n    b: 1\n" {
		t.Errorf("indentBlock blank line = %q", buf.String())
	}
}

func TestRunSetOnlyIsSilent(t *testing.T) {
	f := writeTemp(t, "f.md", "---\nid: 1\n---\nbody\n")

	opts, _ := parseArgs([]string{"--set", ".id=2", f})
	var buf bytes.Buffer
	if err := run(opts, &buf); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no output, got %q", buf.String())
	}
}

func TestRunSetThenGetInOneCall(t *testing.T) {
	f := writeTemp(t, "f.md", "---\nid: 1\n---\nbody\n")

	opts, err := parseArgs([]string{"--set", ".id=2", "--get", "id", f})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := run(opts, &buf); err != nil {
		t.Fatal(err)
	}
	if buf.String() != f+":\n  id: 2\n" {
		t.Errorf("output = %q", buf.String())
	}
}

func TestMultipleSetsAppliedInOrder(t *testing.T) {
	f := writeTemp(t, "f.md", "---\nid: 1\n---\nbody\n")

	opts, err := parseArgs([]string{"--set", ".id=2", "--set", ".id=3", "--set", ".meta.level=4", f})
	if err != nil {
		t.Fatal(err)
	}
	if err := run(opts, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}

	got, _ := os.ReadFile(f)
	want := "---\nid: 3\nmeta:\n  level: 4\n---\nbody\n"
	if string(got) != want {
		t.Errorf("file =\n%s\nwant\n%s", got, want)
	}
}

func TestAllSetsRunBeforeGets(t *testing.T) {
	f := writeTemp(t, "f.md", "---\na: 1\n---\nbody\n")

	opts, err := parseArgs([]string{
		"--set", ".b=2",
		"--get", "a",
		"--set", ".c.d=3",
		"--get", "b",
		"--get", ".c.d",
		f,
	})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := run(opts, &buf); err != nil {
		t.Fatal(err)
	}
	want := f + ":\n  a: 1\n  b: 2\n  c:\n    d: 3\n"
	if buf.String() != want {
		t.Errorf("output =\n%s\nwant\n%s", buf.String(), want)
	}
}

func TestMainRun(t *testing.T) {
	if code := mainRun([]string{}, &bytes.Buffer{}, &bytes.Buffer{}); code != 2 {
		t.Errorf("no args: code=%d want 2", code)
	}

	var help bytes.Buffer
	if code := mainRun([]string{"--help"}, &help, &bytes.Buffer{}); code != 0 {
		t.Errorf("help: code=%d want 0", code)
	}
	if !strings.Contains(help.String(), "Usage:") {
		t.Errorf("help = %q", help.String())
	}

	var stderr bytes.Buffer
	if code := mainRun([]string{"--get", "id", "nope.md"}, &bytes.Buffer{}, &stderr); code != 1 {
		t.Errorf("missing file: code=%d want 1", code)
	}
	if !strings.Contains(stderr.String(), "fmpath:") {
		t.Errorf("stderr = %q", stderr.String())
	}

	f := writeTemp(t, "f.md", "---\nid: 7\n---\nbody\n")
	var stdout bytes.Buffer
	if code := mainRun([]string{"--get", "id", f}, &stdout, &bytes.Buffer{}); code != 0 {
		t.Errorf("get: code=%d want 0", code)
	}
	if stdout.String() != f+":\n  id: 7\n" {
		t.Errorf("stdout = %q", stdout.String())
	}

	if code := mainRun([]string{"--set", ".id=8", f}, &bytes.Buffer{}, &bytes.Buffer{}); code != 0 {
		t.Errorf("set: code=%d want 0", code)
	}
	data, _ := os.ReadFile(f)
	if !strings.Contains(string(data), "id: 8") {
		t.Errorf("file = %q", data)
	}
}
