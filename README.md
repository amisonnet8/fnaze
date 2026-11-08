# fnaze

fnaze writes, reads and checks description files: for one set of changes, one line per changed file saying what changed and why. A reviewer reads the list next to the diff.

- One description file describes one set of changes. What that set is (usually one commit) is up to you
- fnaze does not read or run git, or any other program
- Descriptions can be kept in several languages, one file per language

Not implemented yet. See:

- [docs/reference/file-format.md](docs/reference/file-format.md): where the files are and how they are written
- [docs/reference/commands.md](docs/reference/commands.md): the commands
- [docs/examples/](docs/examples/): example files and a git pre-commit hook

## License

MIT
