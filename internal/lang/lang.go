// Package lang checks the names of languages and reads FNAZE_LANG
// (docs/reference/file-format.md, "File name"; docs/reference/commands.md,
// "Common"). To fnaze a language is just a string: only "en" is special, and
// only in file names (package name).
package lang

import (
	"fmt"
	"strings"
)

// Default is the language used when none is given and FNAZE_LANG is not set.
const Default = "en"

// Check returns an error unless lang is one or more ASCII letters (upper or
// lower case) and digits.
func Check(lang string) error {
	if lang == "" {
		return fmt.Errorf("a language cannot be empty")
	}
	for _, c := range []byte(lang) {
		if !isLetterOrDigit(c) {
			return fmt.Errorf("%q is not a language: use ASCII letters and digits", lang)
		}
	}
	return nil
}

func isLetterOrDigit(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9'
}

// CheckGroup returns an error unless langs can be the languages of one group:
// each is a language, none is given twice, and no two differ only in case
// (their file names would be one file where case is ignored).
func CheckGroup(langs []string) error {
	for i, l := range langs {
		if err := Check(l); err != nil {
			return err
		}
		for _, m := range langs[:i] {
			switch {
			case l == m:
				return fmt.Errorf("%s is given twice", l)
			case strings.EqualFold(l, m):
				return fmt.Errorf("%s and %s differ only in case, so they cannot be in one group", m, l)
			}
		}
	}
	return nil
}

// FromEnv returns the languages in the value of FNAZE_LANG: separated by ",",
// with the spaces around each name and the empty names left out. They are not
// checked.
func FromEnv(value string) []string {
	var langs []string
	for l := range strings.SplitSeq(value, ",") {
		if l = strings.TrimSpace(l); l != "" {
			langs = append(langs, l)
		}
	}
	return langs
}

// One returns the language of a command that takes one: given, or else the
// first in FNAZE_LANG, or else Default.
func One(given, env string) (string, error) {
	l := given
	if l == "" {
		l = Default
		if langs := FromEnv(env); len(langs) > 0 {
			l = langs[0]
		}
	}
	return l, Check(l)
}
