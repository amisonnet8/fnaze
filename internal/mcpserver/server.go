// Package mcpserver is the MCP server of fnaze (fnaze mcp,
// docs/reference/commands.md): each tool calls internal/core and returns the
// same JSON as the CLI's --json (core.JSON), and an error with the same
// message, so that the two never behave differently.
package mcpserver

import (
	"context"
	"io"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/amisonnet8/fnaze/internal/core"
)

// Options are what the server runs with.
type Options struct {
	Dir        string // where to look for .fnaze/ from: -C, or "."
	EnvLang    string // the value of FNAZE_LANG when the server started
	AllowWrite bool   // --allow-write: add the tools that write description files
	Version    string
	Now        func() time.Time // the clock of new_group; nil for time.Now
	Random     io.Reader        // the randomness of new_group; nil for crypto/rand
}

// The arguments of the tools, the same as those of the commands. A field
// without omitempty is required.

type showIn struct {
	Lang  string `json:"lang,omitempty" jsonschema:"the language of the file; omitted for the first language in FNAZE_LANG, or en"`
	Group string `json:"group,omitempty" jsonschema:"the group: its <datetime>_<UUID>, or a start of it that names only one group; omitted for the newest group"`
}

type statusIn struct {
	Group string `json:"group,omitempty" jsonschema:"the group: its <datetime>_<UUID>, or a start of it that names only one group; omitted for the newest group"`
	All   bool   `json:"all,omitempty" jsonschema:"check every group; a group cannot be given with it"`
}

type logIn struct {
	Lang string `json:"lang,omitempty" jsonschema:"the language of the titles; omitted for the first language in FNAZE_LANG, or en"`
}

type newIn struct {
	Langs []string `json:"langs,omitempty" jsonschema:"the languages, one file each; omitted for the languages of the newest group, or else those in FNAZE_LANG, or en"`
}

type titleIn struct {
	Lang  string `json:"lang,omitempty" jsonschema:"the language of the file; omitted for the first language in FNAZE_LANG, or en"`
	Group string `json:"group,omitempty" jsonschema:"the group: its <datetime>_<UUID>, or a start of it that names only one group; omitted for the newest group"`
	Title string `json:"title" jsonschema:"one line: what this set of changes is"`
}

type setIn struct {
	Lang   string   `json:"lang,omitempty" jsonschema:"the language of the file; omitted for the first language in FNAZE_LANG, or en"`
	Group  string   `json:"group,omitempty" jsonschema:"the group: its <datetime>_<UUID>, or a start of it that names only one group; omitted for the newest group"`
	Path   string   `json:"path" jsonschema:"the path of the changed file, from the root, with /; a directory ends with / (src/auth/)"`
	Desc   string   `json:"desc" jsonschema:"one line: what was changed and why"`
	Detail []string `json:"detail,omitempty" jsonschema:"more lines, one per item (a newline starts another line); omitted keeps the detail, and [] or only empty lines remove it"`
}

type removeIn struct {
	Lang  string `json:"lang,omitempty" jsonschema:"the language of the file; omitted for the first language in FNAZE_LANG, or en"`
	Group string `json:"group,omitempty" jsonschema:"the group: its <datetime>_<UUID>, or a start of it that names only one group; omitted for the newest group"`
	Path  string `json:"path" jsonschema:"the path of the entry to remove, as written in set_description"`
}

// New makes the server with its tools.
func New(o Options) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "fnaze", Version: o.Version}, nil)

	addTool(s, o, &mcp.Tool{
		Name: "show",
		Description: "Show the description file of a group in a language as a tree: the title, and for each changed file " +
			"what was changed and why (desc) and more (detail).",
	}, func(c *core.Core, in showIn) (any, error) {
		return c.Show(in.Lang, in.Group)
	})

	addTool(s, o, &mcp.Tool{
		Name: "status",
		Description: "Check a group (or every group): format errors, an empty title, entries without a desc, " +
			"and files of the group in other languages whose paths differ. An empty list means nothing to report.",
	}, func(c *core.Core, in statusIn) (any, error) {
		return c.Status(in.Group, in.All)
	})

	addTool(s, o, &mcp.Tool{
		Name:        "log",
		Description: "List the groups, newest first, with their datetime (UTC), languages, title and number of entries.",
	}, func(c *core.Core, in logIn) (any, error) {
		return c.Log(in.Lang)
	})

	if !o.AllowWrite {
		return s
	}

	addTool(s, o, &mcp.Tool{
		Name: "new_group",
		Description: "Make a new group: one description file for each language, starting from the paths of the newest file " +
			"in that language without the title, descs and details. Writes only .fnaze/; no project file is touched.",
	}, func(c *core.Core, in newIn) (any, error) {
		return c.New(in.Langs)
	})

	addTool(s, o, &mcp.Tool{
		Name: "set_title",
		Description: "Write the title of a description file: one line saying what the set of changes is. " +
			"Writes only that file in .fnaze/.",
	}, func(c *core.Core, in titleIn) (any, error) {
		return c.Title(in.Lang, in.Group, in.Title)
	})

	addTool(s, o, &mcp.Tool{
		Name: "set_description",
		Description: "Write the entry of one changed file (or directory) in a description file: what was changed and why. " +
			"Writes only that file in .fnaze/; the changed file itself is not touched.",
	}, func(c *core.Core, in setIn) (any, error) {
		return c.Set(in.Lang, in.Group, in.Path, in.Desc, in.Detail, in.Detail != nil)
	})

	addTool(s, o, &mcp.Tool{
		Name: "remove_description",
		Description: "Remove the entry of a path, with its children and details, from a description file. " +
			"The file at the path itself is not deleted.",
	}, func(c *core.Core, in removeIn) (any, error) {
		return c.Rm(in.Lang, in.Group, in.Path)
	})

	return s
}

// addTool adds a tool that opens the root for each call (so that changes to
// .fnaze/ between calls are seen) and returns core.JSON of what run returns.
func addTool[In any](s *mcp.Server, o Options, t *mcp.Tool, run func(c *core.Core, in In) (any, error)) {
	mcp.AddTool(s, t, func(_ context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		c, err := open(o)
		if err != nil {
			return nil, nil, err
		}
		out, err := run(c, in)
		if err != nil {
			return nil, nil, err
		}
		b, err := core.JSON(out)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}}, out, nil
	})
}

// Run serves the tools over stdin and stdout until the client closes them.
func Run(ctx context.Context, o Options) error {
	return New(o).Run(ctx, &mcp.StdioTransport{})
}

func open(o Options) (*core.Core, error) {
	c, err := core.Open(o.Dir, o.EnvLang)
	if err != nil {
		return nil, err
	}
	if o.Now != nil {
		c.Now = o.Now
	}
	if o.Random != nil {
		c.Random = o.Random
	}
	return c, nil
}
