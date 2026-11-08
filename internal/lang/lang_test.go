package lang

import (
	"slices"
	"testing"
)

func TestCheck(t *testing.T) {
	for _, l := range []string{"en", "ja", "JA", "zhHant", "x1", "1", "Z", "a", "z", "A", "0", "9"} {
		if err := Check(l); err != nil {
			t.Errorf("%q: %v", l, err)
		}
	}
	// Not ASCII letters and digits: empty, "-", "_", spaces, non-ASCII letters.
	for _, l := range []string{"", "zh-hant", "zh_hant", " ja", "ja ", "j a", "日本", "\xc3\xa9", "ja,en", "a/b", "a.b"} {
		if err := Check(l); err == nil {
			t.Errorf("%q: no error", l)
		}
	}
	// The characters just outside the ranges a-z, A-Z and 0-9.
	for _, l := range []string{"`", "{", "@", "[", "/", ":"} {
		if err := Check(l); err == nil {
			t.Errorf("%q: no error", l)
		}
	}
}

func TestCheckGroup(t *testing.T) {
	for _, langs := range [][]string{nil, {"en"}, {"en", "ja"}, {"ja", "en"}, {"ja", "zh", "ko"}} {
		if err := CheckGroup(langs); err != nil {
			t.Errorf("%q: %v", langs, err)
		}
	}
	for _, langs := range [][]string{
		{"ja", "JA"},
		{"en", "EN"},
		{"ja", "en", "Ja"},
		{"ja", "ja"},
		{"en", "x-y"},
		{"x-y", "en"},
	} {
		if err := CheckGroup(langs); err == nil {
			t.Errorf("%q: no error", langs)
		}
	}
}

func TestFromEnv(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"ja", []string{"ja"}},
		{"ja,en", []string{"ja", "en"}},
		{" ja , ,en ,", []string{"ja", "en"}},
		{",,", nil},
		{"j a", []string{"j a"}},
	}
	for _, tt := range tests {
		if got := FromEnv(tt.in); !slices.Equal(got, tt.want) {
			t.Errorf("%q: got %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestOne(t *testing.T) {
	tests := []struct {
		given, env, want string
	}{
		{"", "", "en"},
		{"", "ja,en", "ja"},
		{"", " , ko", "ko"},
		{"fr", "ja,en", "fr"},
		{"en", "ja", "en"},
	}
	for _, tt := range tests {
		got, err := One(tt.given, tt.env)
		if err != nil || got != tt.want {
			t.Errorf("One(%q, %q) = %q, %v; want %q", tt.given, tt.env, got, err, tt.want)
		}
	}
	for _, tt := range []struct{ given, env string }{{"zh-hant", ""}, {"", "zh-hant,en"}} {
		if _, err := One(tt.given, tt.env); err == nil {
			t.Errorf("One(%q, %q): no error", tt.given, tt.env)
		}
	}
}
