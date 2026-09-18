# fmpath

`fmpath` is a small command-line tool to **read and write the YAML frontmatter of Markdown files** using simple, dot-separated paths.

It takes one or more `.md` files, applies `--get` / `--set` operations on their frontmatter, and always prints valid YAML. It never touches the Markdown body.

## Install

```sh
go install github.com/JLugagne/fmpath@latest
```

Or build from a clone:

```sh
git clone https://github.com/JLugagne/fmpath
cd fmpath
go build -o fmpath .
```

## Usage

```
fmpath [--get PATH]... [--set PATH=VALUE]... FILE...
```

- `FILE...` is one or more Markdown files. Shell globs work as usual (`*.md`).
- `--get` selects a field to read. It can be repeated.
- `--set` assigns a value to a field. It can be repeated.
- Paths may start with a dot and use `.` to descend into nested mappings: `.meta.author.name`.
- `--help` (or `-h`) prints a usage summary and exits.

## Reading fields

Use one or more `--get` to select fields. The output is a YAML mapping keyed by file name, then by the requested paths:

```sh
fmpath --get id --get name *.md
```

```yaml
f1.md:
  id: XX
  name: XX
f2.md:
  id: YY
  name: YY
```

Nested paths are rendered as nested mappings:

```sh
fmpath --get .meta.author.name post.md
```

```yaml
post.md:
  meta:
    author:
      name: Jane Doe
```

If a path points to a mapping (or any subtree), the whole subtree is returned:

```sh
fmpath --get .meta post.md
```

```yaml
post.md:
  meta:
    author:
      name: Jane Doe
    tags:
      - go
      - cli
```

Requested fields that are missing from a file are simply omitted. Files without frontmatter are omitted too. If nothing matches at all, `{}` is printed (still valid YAML).

### Reading the whole frontmatter

If no `--get` is given, the entire frontmatter of every file is printed:

```sh
fmpath *.md
```

```yaml
f1.md:
  id: XX
  name: Alice
  meta:
    level: 3
f2.md:
  id: YY
```

In this mode `fmpath` does not parse or re-encode the YAML: it prints the original frontmatter block, indented under the file name. Formatting, key order and comments are preserved exactly.

## Writing fields

Use `--set .path=value`. Missing intermediate mappings are created automatically:

```sh
fmpath --set .id=42 --set .meta.level=3 post.md
```

Given:

```markdown
---
title: Hello
---

# Body
```

the file becomes:

```markdown
---
title: Hello
id: 42
meta:
  level: 3
---

# Body
```

Notes:

- Existing key order and comments in the frontmatter are preserved; new keys are appended.
- The Markdown body is never modified, and the file keeps its original permissions.
- Values are interpreted as YAML scalars when possible, so `--set .count=3` stores an integer, `--set .draft=true` a boolean, and `--set .title=Hello` a string. Complex or invalid values fall back to a plain string.
- If a file has no frontmatter, a block is created at the top of the file.
- Arrays are not supported by `--set`.

## Combining `--get` and `--set`

When both are given, **all `--set` operations run first**, then the `--get` selections are resolved against the updated files:

```sh
fmpath --set .id=2 --get id --set .meta.level=3 --get .meta file.md
```

```yaml
file.md:
  id: 2
  meta:
    level: 3
```

The order of `--get` flags controls the order of the fields in the output.

## Path syntax

- `.field` and `field` are equivalent (the leading dot is optional).
- `.a.b.c` descends through nested mappings.
- A path component must not be empty (`.a..b` is invalid).

## Exit codes

| Code | Meaning                                  |
|------|------------------------------------------|
| `0`  | Success                                  |
| `1`  | Runtime error (e.g. unreadable file, invalid frontmatter) |
| `2`  | Usage error (bad flags or no file given) |

## How it works

- Frontmatter is detected as a block delimited by `---` at the very start of the file (`...` also accepted as a closing delimiter). CRLF line endings are handled.
- Reading selections are parsed into a YAML node tree. Writing mutates the node tree and re-serializes it with a 2-space indent.
- Reading the whole frontmatter bypasses YAML parsing entirely and just indents the raw block, which is both faster and formatting-preserving.
- Output is produced with [`gopkg.in/yaml.v3`](https://gopkg.in/yaml.v3) so file names and values are quoted when needed and the result is always valid YAML.

## Development

```sh
go test ./...            # run the tests
go test -cover ./...     # test coverage
go vet ./...
go test -run '^$' -bench . -benchmem ./...   # benchmarks
```
