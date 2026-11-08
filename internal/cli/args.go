package cli

import (
	"fmt"
	"slices"
	"strings"
)

// valueOptions are the options that take a value: "--lang ja" or
// "--lang=ja". Every other option is a flag.
var valueOptions = map[string]bool{
	"-C":       true,
	"--lang":   true,
	"--group":  true,
	"--detail": true,
}

// parsed is a command line after the command name.
type parsed struct {
	opts map[string][]string // the options given, with their values in order ("" for a flag)
	args []string            // the positional arguments, in order
	help bool                // -h or --help was given
}

// has reports whether the option was given.
func (p *parsed) has(name string) bool {
	_, ok := p.opts[name]
	return ok
}

// value is the value of an option, or "" when it is not given.
func (p *parsed) value(name string) string {
	if v := p.opts[name]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// parse reads args for a command that accepts the options allowed. Options
// may come before or after the positional arguments; after "--" everything is
// positional. A value option takes the next argument as it is, even one that
// starts with "-" (so --detail "" and --detail -x work). Only --detail may be
// given more than once.
func parse(args []string, allowed []string) (*parsed, error) {
	p := &parsed{opts: map[string][]string{}}
	rest := args
	for len(rest) > 0 {
		a := rest[0]
		rest = rest[1:]
		if a == "--" {
			p.args = append(p.args, rest...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			p.args = append(p.args, a)
			continue
		}
		if a == "-h" || a == "--help" {
			p.help = true
			continue
		}
		name, value, hasValue := strings.Cut(a, "=")
		if !slices.Contains(allowed, name) {
			return nil, fmt.Errorf("unknown option %s", name)
		}
		if p.has(name) && name != "--detail" {
			return nil, fmt.Errorf("%s is given twice", name)
		}
		if !valueOptions[name] {
			if hasValue {
				return nil, fmt.Errorf("%s takes no value", name)
			}
			p.opts[name] = []string{""}
			continue
		}
		if !hasValue {
			if len(rest) == 0 {
				return nil, fmt.Errorf("%s needs a value", name)
			}
			value = rest[0]
			rest = rest[1:]
		}
		p.opts[name] = append(p.opts[name], value)
	}
	return p, nil
}
