package examples

import (
	"strings"
	"testing"
	"testing/fstest"
)

func mustLoad(t *testing.T) []Example {
	t.Helper()
	exs, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return exs
}

func TestLoadKeepsTheOriginalSixFirst(t *testing.T) {
	exs := mustLoad(t)
	want := []string{"counter", "dropdown", "classbind"}
	for i, slug := range want {
		if exs[i].Slug != slug {
			t.Errorf("index %d: slug %q, want %q", i, exs[i].Slug, slug)
		}
	}
	for _, slug := range []string{"search", "tabs", "transition"} {
		if _, ok := Find(exs, slug); !ok {
			t.Errorf("%s missing", slug)
		}
	}
}

func TestLoadInvariants(t *testing.T) {
	exs := mustLoad(t)
	seen := map[string]string{}
	lastGroup := -1
	for _, ex := range exs {
		if ex.Title == "" || ex.Group == "" || ex.Status == "" || len(ex.Features) == 0 {
			t.Errorf("%s: incomplete metadata %+v", ex.Slug, ex)
		}
		if strings.TrimSpace(ex.Alpine) == "" {
			t.Errorf("%s: empty alpine fragment", ex.Slug)
		}
		if strings.Contains(ex.HxLive, "x-data") {
			t.Errorf("%s: hxlive fragment contains x-data", ex.Slug)
		}
		if ex.HasDemo() != (strings.TrimSpace(ex.HxLive) != "") {
			t.Errorf("%s: status %q but hxlive fragment present=%v", ex.Slug, ex.Status, ex.HxLive != "")
		}
		if !strings.Contains(ex.NotesHTML, "<p>") {
			t.Errorf("%s: notes not rendered", ex.Slug)
		}
		for _, f := range ex.Features {
			if prev, dup := seen[f]; dup {
				t.Errorf("feature %q on both %s and %s", f, prev, ex.Slug)
			}
			seen[f] = ex.Slug
		}
		gi := groupIndex(ex.Group)
		if gi < lastGroup {
			t.Errorf("%s: group %q out of order", ex.Slug, ex.Group)
		}
		lastGroup = gi
	}
}

func groupIndex(g Group) int {
	for i, x := range Groups {
		if x == g {
			return i
		}
	}
	return -1
}

func TestFeaturesCoversEveryCardFeatureOnce(t *testing.T) {
	exs := mustLoad(t)
	feats, err := Features(exs)
	if err != nil {
		t.Fatal(err)
	}
	fromCards := map[string]bool{}
	for _, ex := range exs {
		for _, f := range ex.Features {
			fromCards[f] = true
		}
	}
	fromMatrix := map[string]int{}
	for _, f := range feats {
		fromMatrix[f.Feature]++
		ex, ok := Find(exs, f.Slug)
		if !ok || ex.Title != f.Title || ex.Group != f.Group || ex.Status != f.Status {
			t.Errorf("matrix line %q does not match card %q", f.Feature, f.Slug)
		}
	}
	for f := range fromCards {
		if fromMatrix[f] != 1 {
			t.Errorf("feature %q appears %d times in the matrix", f, fromMatrix[f])
		}
	}
	if len(fromMatrix) != len(fromCards) {
		t.Errorf("matrix has %d features, cards have %d", len(fromMatrix), len(fromCards))
	}
}

func TestByGroupAndLabels(t *testing.T) {
	exs := mustLoad(t)
	total := 0
	for _, g := range Groups {
		if g.Label() == "" {
			t.Errorf("group %q has no label", g)
		}
		total += len(ByGroup(exs, g))
	}
	if total != len(exs) {
		t.Errorf("ByGroup partitions %d of %d", total, len(exs))
	}
}

func TestFragmentAndFind(t *testing.T) {
	exs := mustLoad(t)
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

func fakeFS(withHx bool) fstest.MapFS {
	m := fstest.MapFS{
		"a/alpine.html": {Data: []byte("<div x-data></div>\n")},
		"a/notes.md":    {Data: []byte("note\n")},
	}
	if withHx {
		m["a/hxlive.html"] = &fstest.MapFile{Data: []byte("<div></div>\n")}
	}
	return m
}

func TestLoadRejectsStatusFileMismatch(t *testing.T) {
	if _, err := load(fakeFS(true), []row{{"a", "A", Directive, []string{"x-a"}, None}}); err == nil {
		t.Error("none row with hxlive.html: want error")
	}
	if _, err := load(fakeFS(false), []row{{"a", "A", Directive, []string{"x-a"}, Equivalent}}); err == nil {
		t.Error("demo row without hxlive.html: want error")
	}
	if _, err := load(fakeFS(false), []row{{"a", "A", Directive, []string{"x-a"}, None}}); err != nil {
		t.Errorf("valid none row: %v", err)
	}
}

func TestLoadRejectsDuplicateFeatures(t *testing.T) {
	fsys := fakeFS(true)
	fsys["b/alpine.html"] = &fstest.MapFile{Data: []byte("<div x-data></div>\n")}
	fsys["b/hxlive.html"] = &fstest.MapFile{Data: []byte("<div></div>\n")}
	fsys["b/notes.md"] = &fstest.MapFile{Data: []byte("note\n")}
	rows := []row{
		{"a", "A", Directive, []string{"x-a"}, Equivalent},
		{"b", "B", Directive, []string{"x-a"}, Equivalent},
	}
	if _, err := load(fsys, rows); err == nil {
		t.Error("duplicate feature: want error")
	}
}

func TestFeaturesRejectsDisagreement(t *testing.T) {
	exs := []Example{{Slug: "a", Title: "A", Group: Directive, Features: []string{"x-a"}, Status: Equivalent}}
	if _, err := features(exs, []string{"x-a", "x-b"}); err == nil {
		t.Error("order lists unclaimed feature: want error")
	}
	if _, err := features(exs, []string{}); err == nil {
		t.Error("card claims feature missing from order: want error")
	}
	rows, err := features(exs, []string{"x-a"})
	if err != nil || len(rows) != 1 || rows[0].Slug != "a" {
		t.Errorf("valid: rows=%v err=%v", rows, err)
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

func TestMatrixIsComplete(t *testing.T) {
	exs := mustLoad(t)
	feats, err := Features(exs)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"x-bind", "x-bind (class object)", "x-bind (class, tabs)", "x-cloak", "x-data", "x-effect",
		"x-for", "x-for (filtering)", "x-html", "x-id", "x-if", "x-ignore", "x-init",
		"x-model", "x-model (filtering)", "x-modelable", "x-on (modifiers)", "x-on (.outside)",
		"x-ref", "x-show", "x-teleport", "x-text", "x-transition",
		"$data", "$dispatch", "$el", "$id", "$nextTick", "$refs", "$root", "$store", "$watch",
		"Alpine.bind", "Alpine.data", "Alpine.store",
	}
	if len(feats) != len(want) {
		t.Fatalf("matrix has %d lines, want %d", len(feats), len(want))
	}
	for i, f := range feats {
		if f.Feature != want[i] {
			t.Errorf("line %d: %q, want %q", i, f.Feature, want[i])
		}
	}
	if len(exs) != 28 {
		t.Errorf("got %d cards, want 28", len(exs))
	}
	none := 0
	for _, ex := range exs {
		if !ex.HasDemo() {
			none++
		}
	}
	if none != 5 {
		t.Errorf("got %d none rows, want 5", none)
	}
}
