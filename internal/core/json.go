package core

import (
	"bytes"
	"encoding/json"
)

// JSON is the JSON of a result, as --json prints it and the MCP tools return
// it (docs/reference/commands.md, "JSON"): indented by 2 spaces and ending
// with a newline. HTML characters are kept as they are, since descriptions
// are text for people, not for a web page. Both the CLI and the MCP server
// use it, so that their JSON is the same to the byte.
func JSON(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
