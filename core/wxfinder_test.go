package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDisableHandleFinderPicksNewest(t *testing.T) {
	root := t.TempDir()
	older := filepath.Join(root, "old", "engineConfig.json")
	newer := filepath.Join(root, "new", "engineConfig.json")
	other := filepath.Join(root, "skip", "engineConfig.json")
	writeConfig(t, older, `{"featureSwitch":{"handleFinder":true,"keep":1},"n":1234567890123456789}`)
	writeConfig(t, newer, `{"featureSwitch":{"handleFinder":true,"other":true},"name":"keep"}`)
	writeConfig(t, other, `{"featureSwitch":{"somethingElse":true}}`)

	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(older, past, past); err != nil {
		t.Fatal(err)
	}

	res, err := disableHandleFinderIn([]string{root}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Changed || res.Path != newer {
		t.Fatalf("got %+v", res)
	}
	if _, err := os.Stat(res.Backup); err != nil {
		t.Fatal(err)
	}

	body, err := os.ReadFile(newer)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	for _, part := range []string{`"handleFinder": false`, `"other": true`, `"name": "keep"`} {
		if !strings.Contains(text, part) {
			t.Fatalf("rewritten config = %s", text)
		}
	}
	oldBody, _ := os.ReadFile(older)
	for _, part := range []string{`"handleFinder":true`, `"n":1234567890123456789`} {
		if !strings.Contains(string(oldBody), part) {
			t.Fatalf("older config changed: %s", oldBody)
		}
	}

	again, err := disableHandleFinderIn([]string{root}, false)
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed || again.Path != newer {
		t.Fatalf("second run = %+v", again)
	}
}

func TestDisableHandleFinderMissing(t *testing.T) {
	_, err := disableHandleFinderIn([]string{t.TempDir()}, false)
	if err != errFinderNotFound {
		t.Fatalf("err = %v", err)
	}
}

func writeConfig(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}
