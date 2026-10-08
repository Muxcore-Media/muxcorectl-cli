package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const legacyParentalFixture = `{
  "alice": {
    "max_parental_rating": "PG",
    "blocked_tags": "horror",
    "allowed_tags": "",
    "allow_unrated": false,
    "kids_mode": true,
    "pin_hash": "SECRET-HASH-VALUE-123"
  }
}
`

func setParentalFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "parental.json")
	if content != "" {
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("ADMIN_UI_PARENTAL_FILE", p)
	return p
}

// runParental executes the root command and captures stdout (the commands use
// fmt.Print*/printJSON, which write to os.Stdout).
func runParental(t *testing.T, args ...string) (string, error) {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	root := NewRoot()
	root.SetArgs(append([]string{"users", "parental"}, args...))
	var errBuf bytes.Buffer
	root.SetErr(&errBuf)
	runErr := root.Execute()
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	return string(out), runErr
}

func TestUsersParentalSetFailsAndWritesNothing(t *testing.T) {
	p := setParentalFile(t, legacyParentalFixture)
	before, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(p)

	out, err := runParental(t, "set", "alice", "--max-rating", "R", "--blocked-tags", "x", "--allow-unrated")
	if err == nil {
		t.Fatalf("set must fail, got success (stdout=%q)", out)
	}
	for _, want := range []string{"admin-ui", "/users/{id}/parental", "/users/parental/migrate", "ADR-0031"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
	if strings.Contains(out, "saved") {
		t.Fatalf("set must not report success: %q", out)
	}
	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("parental.json changed:\nbefore=%s\nafter=%s", before, after)
	}
	for _, want := range []string{`"kids_mode": true`, `SECRET-HASH-VALUE-123`} {
		if !strings.Contains(string(after), want) {
			t.Fatalf("lost %q from file", want)
		}
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("set left extra files in %s: %v", dir, entries)
	}
}

func TestUsersParentalSetDoesNotCreateFile(t *testing.T) {
	p := setParentalFile(t, "")
	if _, err := runParental(t, "set", "alice", "--max-rating", "PG"); err == nil {
		t.Fatal("set must fail")
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("set created %s (stat err=%v)", p, err)
	}
}

func TestUsersParentalShowLabelsLegacyAndHidesHash(t *testing.T) {
	setParentalFile(t, legacyParentalFixture)
	for _, args := range [][]string{{"show", "alice"}, {"--json", "show", "alice"}} {
		out, err := runParental(t, args...)
		if err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if !strings.Contains(out, "LEGACY") || !strings.Contains(out, "NON-AUTHORITATIVE") {
			t.Fatalf("%v: output not labelled legacy: %q", args, out)
		}
		if strings.Contains(out, "SECRET-HASH-VALUE-123") {
			t.Fatalf("%v: pin hash leaked: %q", args, out)
		}
	}
	out, _ := runParental(t, "show", "alice")
	for _, want := range []string{"kids_mode:     true", "pin_hash:      present (not shown)", "max_rating:    PG"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
	out, _ = runParental(t, "--json", "show", "alice")
	for _, want := range []string{`"kids_mode": true`, `"pin_hash_present": true`, `"authoritative": false`, `"legacy": true`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in %q", want, out)
		}
	}
}

func TestUsersParentalShowUnknownUser(t *testing.T) {
	setParentalFile(t, legacyParentalFixture)
	out, err := runParental(t, "show", "bob")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "LEGACY") || !strings.Contains(out, "no legacy entry") {
		t.Fatalf("out=%q", out)
	}
}

func TestUsersParentalShowMissingAndCorruptFiles(t *testing.T) {
	setParentalFile(t, "")
	if out, err := runParental(t, "show", "alice"); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("missing file: err=%v out=%q", err, out)
	}
	for name, content := range map[string]string{"garbage": "{not json", "null": "null", "array": "[1]"} {
		setParentalFile(t, content)
		out, err := runParental(t, "show", "alice")
		if err == nil || !strings.Contains(err.Error(), "corrupt") {
			t.Fatalf("%s: err=%v out=%q", name, err, out)
		}
	}
}
