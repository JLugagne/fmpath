package main

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

var benchFrontmatter = []byte("---\n" +
	"id: 1234\n" +
	"name: Example Document\n" +
	"draft: false\n" +
	"date: 2026-09-18\n" +
	"meta:\n" +
	"  level: 3\n" +
	"  author:\n" +
	"    first: Jane\n" +
	"    last: Doe\n" +
	"  tags:\n" +
	"    - go\n" +
	"    - cli\n" +
	"    - yaml\n" +
	"---\n" +
	"# Title\n\nSome body content that is never touched.\n")

func BenchmarkSplitFrontmatter(b *testing.B) {
	b.SetBytes(int64(len(benchFrontmatter)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = splitFrontmatter(benchFrontmatter)
	}
}

func BenchmarkParseFrontmatter(b *testing.B) {
	fm, _, _ := splitFrontmatter(benchFrontmatter)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := parseFrontmatter(fm); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseArgs(b *testing.B) {
	args := []string{"--get", "id", "--get", "name", "--get", ".meta.level", "--set", ".draft=true", "a.md", "b.md"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := parseArgs(args); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMapGet(b *testing.B) {
	fm, _, _ := splitFrontmatter(benchFrontmatter)
	root, err := parseFrontmatter(fm)
	if err != nil {
		b.Fatal(err)
	}
	parts := []string{"meta", "author", "first"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, ok := mapGet(root, parts); !ok {
			b.Fatal("no match")
		}
	}
}

func BenchmarkEncodeNode(b *testing.B) {
	fm, _, _ := splitFrontmatter(benchFrontmatter)
	root, err := parseFrontmatter(fm)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := encodeNode(root); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkFiles(b *testing.B, n int) *options {
	b.Helper()
	dir := b.TempDir()
	files := make([]string, n)
	for i := range files {
		p := filepath.Join(dir, "file"+strconv.Itoa(i)+".md")
		if err := os.WriteFile(p, benchFrontmatter, 0o644); err != nil {
			b.Fatal(err)
		}
		files[i] = p
	}
	args := append([]string{"--get", "id", "--get", "name", "--get", ".meta.level"}, files...)
	opts, err := parseArgs(args)
	if err != nil {
		b.Fatal(err)
	}
	return opts
}

func BenchmarkOutputGets(b *testing.B) {
	for _, n := range []int{1, 10, 100} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			opts := benchmarkFiles(b, n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := outputGets(io.Discard, opts); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkOutputWholeFrontmatter(b *testing.B) {
	for _, n := range []int{1, 10, 100} {
		b.Run(strconv.Itoa(n), func(b *testing.B) {
			opts := benchmarkFiles(b, n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if err := outputWholeFrontmatter(io.Discard, opts.files); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkSetFrontmatter(b *testing.B) {
	sets := []setOp{
		{parts: []string{"id"}, value: "42"},
		{parts: []string{"meta", "level"}, value: "9"},
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := setFrontmatter(benchFrontmatter, sets); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkApplySets(b *testing.B) {
	dir := b.TempDir()
	p := filepath.Join(dir, "file.md")
	opts, err := parseArgs([]string{"--set", ".id=42", "--set", ".meta.level=9", p})
	if err != nil {
		b.Fatal(err)
	}
	sets := opts.sets
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		if err := os.WriteFile(p, benchFrontmatter, 0o644); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if err := applySets(p, sets); err != nil {
			b.Fatal(err)
		}
	}
}
