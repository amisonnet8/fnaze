# Commands

> This is the authoritative version. A Japanese translation is in `commands_ja.md`.

> Not implemented yet. This document is the specification to implement.

The file format is in `file-format.md`. "Group" and "root" in this document have the meanings given there.

## Common

- **Language**: every command that works on a language takes it as an argument
  - `new` takes several languages (`fnaze new en ja`). With none, it uses the languages of the newest group; if there is no group yet, the languages in `FNAZE_LANG`; if that is not set, `en`
  - The other commands take one language (`--lang <language>`). With none, they use `FNAZE_LANG`; if that is not set, `en`
  - `FNAZE_LANG` can hold several languages separated by `,` (`FNAZE_LANG=ja,en`). Spaces around a name and empty names are ignored. `new` uses all of them, and the other commands use the first one
  - A command works only on the language it is given. It does not keep the files of the other languages in step: `status` reports the files of one group whose path lines differ
  - A language that does not have the form in `file-format.md` (ASCII letters and digits) is an error
- **Target group**: when a command that takes a group is given none, it uses the newest group (the last one in name order). A group is given as its `<datetime>_<UUID>`, or a prefix of it long enough to name only one group
  - When there is no group at all, `status` and `log` print nothing (`status` still reports the file names that do not match the form), and `show`, `set` and `rm` fail
- **`-C <dir>`**: every command except `completion` works as if run in `<dir>` (it looks for `.fnaze/` from there upward; `init` creates it there)
- **`--json`**: `new`, `set`, `rm`, `status`, `show` and `log` print their result as JSON (see "JSON" below). The MCP tools return the same JSON
- `-C` and `--json` are not repeated in the usage lines of the commands below
- **Help and version**: `fnaze --help` (and `fnaze <command> --help`) prints the usage, and `fnaze --version` prints the version
- **Exit code**: `0` on success, `1` only when `status --require` reports something, and `2` on any error (a wrong command line included)
- **Text output** (without `--json`): paths are relative to the current directory, so that they can be opened as they are (in JSON they are relative to the root). Nothing is colored. `init`, `set` and `rm` print nothing when they succeed
- **Errors** go to standard error as `fnaze: <message>`
- fnaze writes nothing outside `.fnaze/`. It runs no other program (git included)

## JSON

- Indented by 2 spaces. Paths are relative to the root and separated by `/` on every OS; a directory ends in `/`
- Every key is always present: `""` for an empty string, `[]` for an empty list. `detail` is one string, with newlines as `\n`
- Datetimes are UTC in RFC 3339 (`2026-11-06T04:12:00Z`)
- Errors are not JSON: they go to standard error as without `--json`

| Command | Output |
|---|---|
| `new` | `{"group", "files": [{"lang", "path"}]}` |
| `set` | `{"group", "lang", "path", "desc", "detail"}`: the entry after writing. Without `--json`, `set` prints nothing |
| `rm` | `{"group", "lang", "path", "removed"}`: `removed` is the number of entries removed, children and skeleton lines above included. Without `--json`, `rm` prints nothing |
| `status` | `{"items": [{"group", "file", "line", "kind", "message"}]}`: `kind` is `format`, `empty-title`, `empty-desc` or `lang-mismatch`; `line` is `0` when the item is not about one line |
| `show` | `{"group", "lang", "file", "title", "entries": [{"name", "path", "desc", "detail", "children": [...]}]}` |
| `log` | `{"groups": [{"group", "datetime", "langs", "title", "entries"}]}`: newest first; `entries` is the number of entries |

## `fnaze init`

Creates `.fnaze/changes/` in the current directory (or in `-C <dir>`). Fails if a `.fnaze/` is found there or in any directory above it (a project has only one `.fnaze/`).

## `fnaze new [language…]`

Creates a new group: one file for each language (given, or chosen as below), all with the same datetime and UUID.

```
fnaze new ja           # <datetime>_<UUID>_ja.txt
fnaze new en ja        # <datetime>_<UUID>.txt and <datetime>_<UUID>_ja.txt
fnaze new              # the same languages as the newest group
```

- **With no language**, it uses the languages of the newest group. If there is no group yet, it uses the languages in `FNAZE_LANG`, or `en` if that is not set
- **`en` is created only when it is given**, like any other language
- Fails if two of the languages differ only in case (`ja` and `JA`): see `file-format.md`
- **Content**: for each language, it takes the newest file in that language and removes the title, descs and details
  - Kept: the path lines and their indentation
  - Removed: the title text (only `# ` remains), the descs (with their tabs), and the detail lines
  - If there is no file in that language yet, it creates a file that holds only `# `
- It prints the path of each file it created, one per line, relative to the current directory

```
$ fnaze new en ja
.fnaze/changes/20261108T010203Z_00010203-0405-4607-8809-0a0b0c0d0e0f.txt
.fnaze/changes/20261108T010203Z_00010203-0405-4607-8809-0a0b0c0d0e0f_ja.txt
```

Example: if the newest `ja` file is

```
# 認証の期限を延長に変えた
src/
	auth/	期限を延長に変えた
		session.go	アクセスのたびに期限を延ばす
			| 短すぎてセッションが切れる報告があった
```

then `fnaze new ja` creates

```
# 
src/
	auth/
		session.go
```

## `fnaze set [--lang <language>] [--group <group>] <path> <desc> [--detail <detail>]…`

Writes one entry to the target group's file in the language (see "Language" under "Common").

- `--group <group>` names the group (default: the newest group)
- The path is taken as written, from the root: it is a directory when it ends with `/`. It must be a path that `file-format.md` allows. A desc cannot have a newline
- If an entry for the path exists, its desc is replaced, and its detail too when `--detail` is given
- If not, the entry is added under the deepest directory entry above it in the file, after its siblings. Missing intermediate directory entries are added as skeleton lines, one per level
- Each `--detail` is one line of the detail, and a newline in a value starts another line. If every value is empty, the detail is removed
- Fails if the group has no file in that language, or the file has a format error
- The file is written again in the canonical form (`file-format.md`, "Content")
- The group's files in other languages are not changed (`status` reports the path lines that no longer match)

## `fnaze rm [--lang <language>] [--group <group>] <path>`

Removes one entry, with its children and details, from the target group's file in the language. `--group` works as in `set`.

- Directory entries above it that are left with no children and have neither a desc nor a detail (skeleton lines) are removed too
- Fails if the group has no file in that language, the file has a format error, or the file has no entry for the path
- The file is written again in the canonical form

## `fnaze status [--require] [--all | group]`

Checks a group, looking at all of its language files.

It reports:

| Kind | Meaning |
|---|---|
| Format error | No title line, broken indentation, the same path twice, a path that is not allowed, a file entry with children, a BOM, a file in `.fnaze/changes/` whose name does not match the form |
| Empty title | A title that is only `#` or `# ` |
| Empty desc | An entry without children and without a desc |
| Language mismatch | Files of one group whose path lines (the paths and their order) differ |

- It prints nothing when there is nothing to report. Otherwise it prints one line for each item, in the order of the files and then of the lines, as `<file>:<line>: <kind>: <message>` (`:<line>` is left out when the line is 0). The file is relative to the current directory, and the kind is the one in JSON

```
.fnaze/changes/20261108T010203Z_00010203-0405-4607-8809-0a0b0c0d0e0f.txt:3: empty-desc: src/a.go has no desc
.fnaze/changes/notes.txt: format: the name does not have the form <datetime>_<UUID>[_<language>].txt
```

- Without `--require`, the exit code is 0 whatever is reported. With `--require`, it is 1 if anything is reported. An error is 2, as in every command
- `--all` checks every group. A group cannot be given with it
- A file in `.fnaze/changes/` whose name does not match the form is reported whichever group is checked (its `group` is `""` in JSON)

## `fnaze show [--lang <language>] [group]`

Shows the group's file in the language as a tree: the title first, the paths and descs in aligned columns, and the details under their desc. Fails if the group has no file in that language.

- The title is followed by an empty line. Each level is indented by two spaces, and the paths are written as in the file
- The descs start in one column: two spaces after the widest path, counted in display width (full width is 2)
- Each detail line is shown in the column of the descs, after `| `

```
# Extend the session lifetime on access instead of fixing it

src/
  auth/           Change the session lifetime from fixed to extended
    session.go    Extend the lifetime on every access
    *_test.go     Add tests for the extension
config/auth.yaml  Set the default session.ttl to 30m
                  | Users reported sessions expiring too soon
docs/old.md       Removed; its content moved to docs/auth.md
```

## `fnaze log [--lang <language>]`

Lists the groups, newest first, one per line: the group's short name, the datetime, the languages the group has, the number of entries, and the title in the language. A group with no file in that language is shown with an empty title. The datetime is shown in the local time zone to the minute (UTC in JSON).

- The short name is the start of the group, long enough to tell it from the other groups: at least the datetime, `_` and four characters of the UUID. It can be given wherever a group is taken
- The languages are joined with `,`. The columns are aligned

```
20261107T021530Z_2c4c  2026-11-07 11:15  en,ja   3  Log session extensions
20261107T003426Z_8980  2026-11-07 09:34  en,ja   6  Extend the session lifetime on access instead of fixing it
```

(The example is in `TZ=Asia/Tokyo`.)

## `fnaze mcp [--allow-write]`

Runs as an MCP server over standard input and output, so AI agents can call fnaze as a tool.

- The server has no language of its own. Each tool takes its language as an argument, in the same way as the command (`new`: several languages; the others: one), and when it is left out, the language is chosen as for the command
- Each tool returns the same JSON as the command with `--json`. An error is returned as a tool error, with the same message as the command
- Read tools have the names of the commands: `show`, `status` and `log`
- Write tools have names of their own, so that an agent does not take them for tools that create or delete files: `new_group` (`new`), `set_description` (`set`) and `remove_description` (`rm`). Offered only with `--allow-write`

## `fnaze completion <shell>`

Prints a shell completion script. The shell is `bash`, `zsh`, `fish` or `powershell`.

```
fnaze completion bash >~/.local/share/bash-completion/completions/fnaze
fnaze completion zsh >"${fpath[1]}/_fnaze"
fnaze completion fish >~/.config/fish/completions/fnaze.fish
fnaze completion powershell >>$PROFILE
```

It completes:

- Command names and their flags
- Groups (`--group` of `set` and `rm`, the argument of `status` and `show`): the `<datetime>_<UUID>` of each existing group, described by its title in the language
- Languages (the arguments of `new`, and `--lang`): the languages used by the existing groups
- Paths (`<path>` of `set` and `rm`): the entry paths in the target group's file in the language

The language here is the `--lang` already on the command line being completed, or the first language in `FNAZE_LANG`, or `en`.

Groups, languages and paths are found by calling fnaze on each completion, which reads `.fnaze/` (no other program is run). `.fnaze/` is looked for from the `-C <dir>` on the command line, if there is one.
