# File format

> This is the authoritative version. A Japanese translation is in `file-format_ja.md`.

Where fnaze keeps its description files, how they are named, and how they are written.

## Location

```
.fnaze/
	changes/
		<datetime>_<UUID>.txt              ← en
		<datetime>_<UUID>_<language>.txt   ← other than en
```

- `fnaze init` creates `.fnaze/`. fnaze searches upward from the current directory (or the one given with `-C`) and uses the first `.fnaze/` it finds. There is one `.fnaze/` per project; they are not nested (`fnaze init` fails under an existing one). This document calls the directory that holds `.fnaze/` the "root"
- **One description file describes one set of changes.** The user decides what that set is (one commit, for example)

## File name

```
<datetime>_<UUID>[_<language>].txt
```

| Part | Form | Content |
|---|---|---|
| datetime | `YYYYMMDDTHHMMSSZ` (16 characters) | When `fnaze new` created the file (UTC) |
| UUID | `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx` (36 characters, lowercase hex) | A random UUID that `fnaze new` makes each time |
| language | One or more of letters, digits, `-`, `_` | Any string. For `en`, it is left out together with its `_` |

- **Sorting the names (by byte) puts the files in the order they were created.** Files with the same datetime are ordered by UUID
- **Files that share the same `<datetime>_<UUID>`** are the descriptions of one set of changes, one per language. This document calls them a "group"
- **Reading the language**: the datetime and the UUID at the start have fixed lengths, so the language is everything from the `_` after them up to `.txt`. It may contain `_` (the language of `…_zh_hant.txt` is `zh_hant`)
- **To fnaze, a language is just a string.** fnaze does not know that `ja` means Japanese. Only `en` is special: it has no suffix
- A name ending in `_en.txt` is not allowed (`en` is written without a suffix)
- Files in `.fnaze/changes/` whose names do not match this form are not read (`status` reports them)

Example:

```
.fnaze/changes/
	20261107T003426Z_8980a325-8b57-49b0-8e0a-ed0aaf9eeda7.txt
	20261107T003426Z_8980a325-8b57-49b0-8e0a-ed0aaf9eeda7_ja.txt
	20261107T021530Z_2c4cc818-64d6-45e8-9f3a-5471dba04be7.txt
	20261107T021530Z_2c4cc818-64d6-45e8-9f3a-5471dba04be7_ja.txt
```

## Content

UTF-8 text with LF line endings. The first line is the title. Each following line is an entry or a detail.

```
# Extend the session lifetime on access instead of fixing it
src/
	auth/	Change the session lifetime from fixed to extended
		session.go	Extend the lifetime on every access
		*_test.go	Add tests for the extension
config/auth.yaml	Set the default session.ttl to 30m
	| Users reported sessions expiring too soon
docs/old.md	Removed; its content moved to docs/auth.md
```

(Both the indentation and the separator between a path and its desc are tabs.)

### Title (first line)

- Starts with `# `, followed by one line saying what this set of changes is
- Written in the file's language
- A line that is only `#` or `# ` is an empty title

### Entries

```
<indent><path>[<tab><desc>]
```

- **Indentation** is tabs only, one tab per level. A line cannot be more than one level deeper than the entry before it
- **Paths** are separated by `/`. A directory ends with `/`
  - The path of a top-level entry is relative to the root
  - The path of an indented entry is joined to the path of the nearest less-indented directory entry above it (`session.go` in the example is `src/auth/session.go`)
  - A path may contain `/`, so the lines for intermediate directories can be left out (`config/auth.yaml` in the example)
  - Only directory entries can have indented children
  - `.`, `..`, empty parts (`a//b`) and a leading `/` are not allowed
- **A desc** is one line. Its length has no limit
- **An entry without a desc**:
  - If it is a directory entry with children, it is a skeleton line for writing deeper paths (`src/` in the example)
  - If it has no children, it has an **empty desc** (something left unwritten)
- **One file cannot have two entries with the same path**
- **Whether a path exists is not checked.** Deleted files and the old names of renamed files can be written (`docs/old.md` in the example)

### Patterns

- The last part of a path may contain `*`. `*` matches any run of characters other than `/`, including none
- A pattern matches only directly inside the directory where it is written (`*_test.go` in the example matches directly inside `src/auth/`, not inside `src/auth/x/`)
- Directory patterns are allowed (`gen-*/`)

### Directory desc

A directory's desc describes the changes under it as a whole. For a large rename or a mechanical rewrite, the files do not have to be written one by one.

To find the description of a path, the closest one is used:

1. The entry for that path
2. A pattern that matches that path
3. The nearest ancestor directory entry that has a desc

### Details

```
<indent>| <detail>
```

- A note on the entry just before it (a reason, background, and so on). It is indented one level deeper than that entry
- An entry may have several detail lines in a row. They are read in order as one note
- Its length has no limit

### Files of one group

- The files of one group have **the same path lines in the same order**. Only the words of the title, descs and details differ
- fnaze does not decide which languages a group should have (each group has the languages that `fnaze new` created it with)
