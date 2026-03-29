package tomlsawyer

import (
	"testing"
)

func TestPruneEmptySection(t *testing.T) {
	doc, err := ParseString("[empty]\n\n[notempty]\nx = 1\n")
	if err != nil {
		t.Fatal(err)
	}
	doc.Prune()
	wantGolden(t, doc.String(), "[notempty]\nx = 1\n")
}

func TestPruneAfterDelete(t *testing.T) {
	doc, err := ParseString("[section]\na = 1\nb = 2\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Delete("section.a"); err != nil {
		t.Fatal(err)
	}
	if err := doc.Delete("section.b"); err != nil {
		t.Fatal(err)
	}
	doc.Prune()
	wantGolden(t, doc.String(), "")
}

func TestPruneAfterMove(t *testing.T) {
	doc, err := ParseString("[src]\nkey = \"val\"\n\n[dst]\nother = 1\n")
	if err != nil {
		t.Fatal(err)
	}
	if err := doc.Move("src.key", "dst.key"); err != nil {
		t.Fatal(err)
	}
	doc.Prune()

	if doc.Has("src") {
		t.Error("expected [src] to be pruned")
	}
	got, err := doc.Get("dst.key")
	if err != nil {
		t.Fatal(err)
	}
	if got != "val" {
		t.Errorf("dst.key = %v, want %q", got, "val")
	}
}

func TestPruneKeepsSectionWithChildren(t *testing.T) {
	doc, err := ParseString("[parent]\n\n[parent.child]\nx = 1\n")
	if err != nil {
		t.Fatal(err)
	}
	doc.Prune()

	if !doc.Has("parent.child.x") {
		t.Error("expected parent.child.x to exist")
	}
	// [parent] should be kept because [parent.child] is a child
	got := doc.String()
	if got != "[parent]\n\n[parent.child]\nx = 1\n" {
		t.Errorf("unexpected output:\n%s", got)
	}
}

func TestPruneKeepsNonEmpty(t *testing.T) {
	input := "[a]\nx = 1\n\n[b]\ny = 2\n"
	doc, err := ParseString(input)
	if err != nil {
		t.Fatal(err)
	}
	doc.Prune()
	wantGolden(t, doc.String(), input)
}

func TestPruneCommentsOnlySection(t *testing.T) {
	doc, err := ParseString("[comments]\n# just a comment\n\n[real]\nx = 1\n")
	if err != nil {
		t.Fatal(err)
	}
	doc.Prune()
	wantGolden(t, doc.String(), "[real]\nx = 1\n")
}

func TestPruneMultipleEmpty(t *testing.T) {
	doc, err := ParseString("[a]\n\n[b]\n\n[c]\nx = 1\n\n[d]\n")
	if err != nil {
		t.Fatal(err)
	}
	doc.Prune()
	wantGolden(t, doc.String(), "[c]\nx = 1\n")
}
