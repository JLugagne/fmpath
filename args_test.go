package main

import "testing"

func TestParsePath(t *testing.T) {
	tests := []struct {
		in      string
		want    []string
		wantErr bool
	}{
		{".field", []string{"field"}, false},
		{"field", []string{"field"}, false},
		{".root.level.key", []string{"root", "level", "key"}, false},
		{"", nil, true},
		{".", nil, true},
		{".a..b", nil, true},
	}
	for _, tt := range tests {
		got, err := parsePath(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parsePath(%q) expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parsePath(%q) unexpected error: %v", tt.in, err)
			continue
		}
		if len(got) != len(tt.want) {
			t.Fatalf("parsePath(%q) = %v, want %v", tt.in, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("parsePath(%q) = %v, want %v", tt.in, got, tt.want)
			}
		}
	}
}

func TestParseArgs(t *testing.T) {
	opts, err := parseArgs([]string{"--get", "id", "--get=name", "--set", ".x=1", "a.md", "b.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.gets) != 2 || opts.gets[0].raw != "id" || opts.gets[1].raw != "name" {
		t.Fatalf("gets = %+v", opts.gets)
	}
	if len(opts.sets) != 1 || opts.sets[0].value != "1" {
		t.Fatalf("sets = %+v", opts.sets)
	}
	if len(opts.files) != 2 {
		t.Fatalf("files = %v", opts.files)
	}
}

func TestParseArgsErrors(t *testing.T) {
	if _, err := parseArgs([]string{"--get", "id"}); err == nil {
		t.Error("expected error when no file given")
	}
	if _, err := parseArgs([]string{"--set", "novalue", "a.md"}); err == nil {
		t.Error("expected error on set without =")
	}
	if _, err := parseArgs([]string{"--bogus", "a.md"}); err == nil {
		t.Error("expected error on unknown flag")
	}
}

func TestParseArgsFilesOnly(t *testing.T) {
	opts, err := parseArgs([]string{"a.md", "b.md"})
	if err != nil {
		t.Fatal(err)
	}
	if len(opts.gets) != 0 || len(opts.sets) != 0 || len(opts.files) != 2 {
		t.Fatalf("opts = %+v", opts)
	}
}
