// Package name reads and makes the names of description files
// (docs/reference/file-format.md, "File name"):
//
//	<datetime>_<UUID>[_<language>].txt
//
// The datetime and the UUID, joined by "_", name the group. The language "en"
// has no suffix.
package name

import (
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/amisonnet8/fnaze/internal/lang"
)

const (
	datetimeLayout = "20060102T150405Z"
	datetimeLen    = len(datetimeLayout)
	uuidLen        = 36
	// GroupLen is the length of a group: <datetime>_<UUID>.
	GroupLen = datetimeLen + 1 + uuidLen
	ext      = ".txt"
)

// Name is the name of a description file.
type Name struct {
	Group string // <datetime>_<UUID>
	Lang  string
}

// File returns the file name.
func (n Name) File() string {
	if n.Lang == lang.Default {
		return n.Group + ext
	}
	return n.Group + "_" + n.Lang + ext
}

// Parse reads a file name. It reports false when the name does not have the
// form (such a file is not read, and status reports it).
func Parse(file string) (Name, bool) {
	rest, ok := strings.CutSuffix(file, ext)
	if !ok || len(rest) < GroupLen || !IsGroup(rest[:GroupLen]) {
		return Name{}, false
	}
	n := Name{Group: rest[:GroupLen], Lang: lang.Default}
	if suffix := rest[GroupLen:]; suffix != "" {
		l, ok := strings.CutPrefix(suffix, "_")
		if !ok || l == lang.Default || lang.Check(l) != nil {
			return Name{}, false
		}
		n.Lang = l
	}
	return n, true
}

// IsGroup reports whether s is a whole group: a datetime in UTC, "_", and a
// UUID in lower-case hex.
func IsGroup(s string) bool {
	if len(s) != GroupLen || s[datetimeLen] != '_' {
		return false
	}
	if _, err := time.Parse(datetimeLayout, s[:datetimeLen]); err != nil {
		return false
	}
	u := s[datetimeLen+1:]
	for i := range len(u) {
		switch c := u[i]; {
		case i == 8 || i == 13 || i == 18 || i == 23:
			if c != '-' {
				return false
			}
		case !isLowerHex(c):
			return false
		}
	}
	return true
}

func isLowerHex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f'
}

// Datetime returns when the group was made. The group must be one that
// IsGroup accepts.
func Datetime(group string) time.Time {
	t, _ := time.Parse(datetimeLayout, group[:datetimeLen])
	return t
}

// NewGroup makes the group of a new set of files: now in UTC, and a random
// UUID (version 4) read from random.
func NewGroup(now time.Time, random io.Reader) (string, error) {
	var b [16]byte
	if _, err := io.ReadFull(random, b[:]); err != nil {
		return "", fmt.Errorf("making a UUID: %w", err)
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	h := hex.EncodeToString(b[:])
	uuid := h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
	return now.UTC().Format(datetimeLayout) + "_" + uuid, nil
}
