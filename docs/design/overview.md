*[日本語](overview_ja.md) | **English***

# The design of fnaze

## What it is for

When AI writes code, a human has to review the changes. The changes themselves do not say why each file was changed, and a commit message speaks for the whole set at once. When many files were changed, nothing says where to start.

With fnaze, **for each set of changes, you make a file that says in one line, for each changed file, what was changed and why** (a description file). A reviewer reads this list to see, file by file, what was changed and why.

## Goals

- A human can see the changes of one set, one line of reason per file: reading this list is enough to know where to look and why each file was changed
- The writer (a human or an AI) only writes once, at the end of the set. The way files are edited does not change
- What was left out (an empty title or desc, files of one group that differ between languages) can be checked by a tool

## What it does not do

- **Work with git or any other version control.** fnaze does not read or run git. It only makes, reads and checks description files
- Find the list of changed files. The writer decides what to write
- Find the kind of change (added, modified, deleted, renamed). If it matters, say it in words in the desc
- Keep the diff itself
- Record edit operations, or the commands that were run
- Record design decisions or todos
- Describe the current structure of the project (fnaze describes only "changes")
- Force how sets are divided, or force checks. For those who want to stop a commit, it only shows an example setup (`docs/examples/`)

## Design rules

### One description file is one set of changes

Which changes it refers to (one commit, several commits, work in progress) is not decided by fnaze. The user decides. The usual way is to add one description file per commit.

### No version control

fnaze reads nothing but the description files. Its check (`status`) is closed within the description files too.

- So fnaze cannot know about "a path that was changed but has no description". Those who want to stop that compare the changes in git with the description files in their own hook (a git pre-commit hook, for example)
- It does not mark the kind of change (A, M, D, R) either: even with a field for it, fnaze could not check that it is right

### File names are "datetime_UUID"

`.fnaze/changes/<datetime>_<UUID>_<language>.txt` (only `en` has no suffix).

- With the **datetime (UTC)**, the name alone gives the order. It can be read without the history of version control
- With the **UUID**, files made in the same second in different places (branches, for example) do not get the same name. Serial numbers would give the same number on different branches
- **The files of one group, one per language, are made together by `fnaze new` so that their names match.** A datetime taken for each language would differ by the time between them, and the names would not match
- **Commit hashes are not used.** They are not known until the commit is made

### A language name is only a string

fnaze does not know that `ja` is Japanese. A language name is only a string that separates the files of each language. Only `en` is special: its file name has no suffix.

- `en` is not required either. `fnaze new` makes files only for the languages given (when none are given, those of the previous group)
- **Every command that works on a language takes it as an argument** (`init` and `completion` do not work on languages; `mcp` takes it for each tool, as below). `new` makes a group, so it takes several languages; the others take one. When none is given, `new` uses the languages of the previous group, and the others use `FNAZE_LANG`, or else `en`. One person usually keeps writing in the same language, so it can usually be left out
- **`FNAZE_LANG` can hold several languages** (`ja,en`). `new` uses all of them, and the other commands use the first one
- **Commands do not keep the files of the languages in step.** `status` reports where they differ, so the other commands work only on the file of the language given, and stay simple
- **`mcp` has no language of its own.** Each tool takes the language as an argument, so one server can write descriptions in several languages

### A new file starts from the skeleton of the previous file

`fnaze new` makes each file from the previous file in the same language, without its title, descs and details (only the path lines). The same places are often changed again, so the paths need not be typed again. The writer removes the lines that are not needed.

### How the file is written

The paths form a tree with tab indentation, and a tab separates a path from its desc. The desc of a directory sums up the changes under it. See `docs/reference/file-format.md`.
