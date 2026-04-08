package bucket_test

import (
	"testing"

	"github.com/bottlesdevs/next-deps-srv/internal/bucket"
)

func TestChar(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"kernel32.dll", "k"},
		{"d3d11.dll", "d"},
		{"-test.bin", "-"},
		{"NTDLL.DLL", "n"},
		{"123.exe", "1"},
		{"@symbol.dll", "-"},
	}
	for _, tc := range cases {
		got := bucket.Char(tc.name)
		if got != tc.want {
			t.Errorf("Char(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestBucketPath(t *testing.T) {
	root := "/data/bucket"
	path := bucket.BucketPath(root, "kernel32.dll")
	if path == "" {
		t.Error("BucketPath returned empty string")
	}
	if path != "/data/bucket/k/kernel32.dll" {
		t.Errorf("unexpected path: %s", path)
	}
}

func TestAllChars(t *testing.T) {
	chars := bucket.AllChars()
	if len(chars) == 0 {
		t.Error("AllChars returned empty")
	}
	seen := make(map[string]bool)
	for _, c := range chars {
		if seen[c] {
			t.Errorf("duplicate char: %q", c)
		}
		seen[c] = true
	}
	if !seen["-"] {
		t.Error("AllChars missing '-' symbol bucket")
	}
}
