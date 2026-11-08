package name

import (
	"bytes"
	"errors"
	"testing"
	"testing/iotest"
	"time"
)

const group = "20261107T003426Z_8980a325-8b57-49b0-8e0a-ed0aaf9eeda7"

func TestParse(t *testing.T) {
	tests := []struct {
		file string
		want Name
	}{
		{group + ".txt", Name{group, "en"}},
		{group + "_ja.txt", Name{group, "ja"}},
		{group + "_JA.txt", Name{group, "JA"}},
		{group + "_zhHant2.txt", Name{group, "zhHant2"}},
		{group + "_EN.txt", Name{group, "EN"}},
	}
	for _, tt := range tests {
		got, ok := Parse(tt.file)
		if !ok || got != tt.want {
			t.Errorf("%s: got %+v, %v", tt.file, got, ok)
		}
		if f := got.File(); f != tt.file {
			t.Errorf("%+v: File() = %s, want %s", got, f, tt.file)
		}
	}
}

func TestParseNotTheForm(t *testing.T) {
	for _, file := range []string{
		group + "_en.txt",           // en has no suffix
		group + "_.txt",             // empty language
		group + "_zh_hant.txt",      // "_" in a language
		group + "_zh-hant.txt",      // "-" in a language
		group + "ja.txt",            // no "_" before the language
		group,                       // no .txt
		group + ".TXT",              // .txt in another case
		group + ".txt.bak",          // more after .txt
		group[:GroupLen-1] + ".txt", // UUID too short
		"20261107T003426Z_8980A325-8b57-49b0-8e0a-ed0aaf9eeda7.txt", // upper-case hex
		"20261107T003426Z_8980a325-8b57-49b0-8e0a-ed0aaf9eeda.txt",  // UUID one short
		"20261107T003426Z_8980a325x8b57-49b0-8e0a-ed0aaf9eeda7.txt", // no "-"
		"20261107T003426Z_8980a3258-b57-49b0-8e0a-ed0aaf9eeda7.txt", // "-" in the wrong place
		"20261107T003426Z_8980a325-8b57-49b0-8e0a-ed0aaf9eedg7.txt", // g is not hex
		"20261107T003426Z-8980a325-8b57-49b0-8e0a-ed0aaf9eeda7.txt", // "-" for "_"
		"20261307T003426Z_8980a325-8b57-49b0-8e0a-ed0aaf9eeda7.txt", // month 13
		"20261107T243426Z_8980a325-8b57-49b0-8e0a-ed0aaf9eeda7.txt", // hour 24
		"20261107 003426Z_8980a325-8b57-49b0-8e0a-ed0aaf9eeda7.txt", // no T
		"20261107T003426+_8980a325-8b57-49b0-8e0a-ed0aaf9eeda7.txt", // no Z
		".txt",
		"",
		"notes.txt",
	} {
		if n, ok := Parse(file); ok {
			t.Errorf("%q: read as %+v", file, n)
		}
	}
}

func TestIsGroup(t *testing.T) {
	if !IsGroup(group) {
		t.Errorf("%s is not a group", group)
	}
	for _, s := range []string{"", group[:10], group + "0", group + "_ja"} {
		if IsGroup(s) {
			t.Errorf("%q is a group", s)
		}
	}
}

func TestDatetime(t *testing.T) {
	want := time.Date(2026, 11, 7, 0, 34, 26, 0, time.UTC)
	if got := Datetime(group); !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// The UUIDs are from Python: uuid.UUID(bytes=..., version=4). The datetime of
// 09:34:26 at +09:00 is from date: TZ=UTC date -d ... +%Y%m%dT%H%M%SZ.
func TestNewGroup(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	tests := []struct {
		now    time.Time
		random []byte
		want   string
	}{
		{time.Date(2026, 11, 7, 9, 34, 26, 0, jst), []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}, "20261107T003426Z_00010203-0405-4607-8809-0a0b0c0d0e0f"},
		{time.Date(2026, 11, 7, 0, 34, 26, 999, time.UTC), bytes.Repeat([]byte{0xff}, 16), "20261107T003426Z_ffffffff-ffff-4fff-bfff-ffffffffffff"},
	}
	for _, tt := range tests {
		got, err := NewGroup(tt.now, bytes.NewReader(tt.random))
		if err != nil || got != tt.want {
			t.Errorf("got %q, %v; want %q", got, err, tt.want)
		}
		if !IsGroup(got) {
			t.Errorf("%s is not a group", got)
		}
	}
}

func TestNewGroupRandomFails(t *testing.T) {
	failure := errors.New("no randomness")
	if _, err := NewGroup(time.Now(), iotest.ErrReader(failure)); !errors.Is(err, failure) {
		t.Errorf("got %v", err)
	}
	if _, err := NewGroup(time.Now(), bytes.NewReader(make([]byte, 15))); err == nil {
		t.Error("15 random bytes: no error")
	}
}
