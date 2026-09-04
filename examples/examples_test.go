package examples

import (
	"strings"
	"testing"
)

func TestLoadReturnsSixExamplesInOrder(t *testing.T) {
	exs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"counter", "dropdown", "search", "tabs", "transition", "classbind"}
	if len(exs) != len(want) {
		t.Fatalf("got %d examples, want %d", len(exs), len(want))
	}
	for i, ex := range exs {
		if ex.Slug != want[i] {
			t.Errorf("index %d: slug %q, want %q", i, ex.Slug, want[i])
		}
		if ex.Title == "" {
			t.Errorf("%s: empty title", ex.Slug)
		}
		if !strings.Contains(ex.Alpine, "x-data") {
			t.Errorf("%s: alpine fragment lacks x-data", ex.Slug)
		}
		if strings.Contains(ex.HxLive, "x-data") {
			t.Errorf("%s: hxlive fragment contains x-data", ex.Slug)
		}
		if strings.TrimSpace(ex.HxLive) == "" {
			t.Errorf("%s: empty hxlive fragment", ex.Slug)
		}
		if !strings.Contains(ex.NotesHTML, "<p>") {
			t.Errorf("%s: notes not rendered to HTML: %q", ex.Slug, ex.NotesHTML)
		}
	}
}

func TestFragmentAndFind(t *testing.T) {
	exs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	ex, ok := Find(exs, "counter")
	if !ok {
		t.Fatal("counter not found")
	}
	if ex.Fragment(Alpine) != ex.Alpine || ex.Fragment(HxLive) != ex.HxLive {
		t.Error("Fragment does not select the right side")
	}
	if _, ok := Find(exs, "nope"); ok {
		t.Error("Find returned ok for unknown slug")
	}
}

func TestParseLib(t *testing.T) {
	for s, want := range map[string]Lib{"alpine": Alpine, "hxlive": HxLive} {
		got, ok := ParseLib(s)
		if !ok || got != want {
			t.Errorf("ParseLib(%q) = %q, %v", s, got, ok)
		}
	}
	if _, ok := ParseLib("react"); ok {
		t.Error("ParseLib accepted react")
	}
	if Alpine.Entry() != "web/frame-alpine.js" || HxLive.Entry() != "web/frame-hxlive.js" {
		t.Error("wrong entry paths")
	}
}
