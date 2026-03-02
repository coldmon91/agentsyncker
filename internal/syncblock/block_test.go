package syncblock

import "testing"

func TestUpsertInsert(t *testing.T) {
	updated, replaced := Upsert("", "~/.claude/CLAUDE.md", "hello")
	if replaced {
		t.Fatal("expected insert, got replace")
	}
	want := "<!-- PROMAN-SYNC-START source=~/.claude/CLAUDE.md -->\nhello\n<!-- PROMAN-SYNC-END -->\n"
	if updated != want {
		t.Fatalf("unexpected output:\n%s", updated)
	}
}

func TestUpsertReplace(t *testing.T) {
	existing := "prefix\n<!-- PROMAN-SYNC-START source=old -->\nold\n<!-- PROMAN-SYNC-END -->\nsuffix\n"
	updated, replaced := Upsert(existing, "new", "content")
	if !replaced {
		t.Fatal("expected replace")
	}
	want := "prefix\n<!-- PROMAN-SYNC-START source=new -->\ncontent\n<!-- PROMAN-SYNC-END -->\nsuffix\n"
	if updated != want {
		t.Fatalf("unexpected output:\n%s", updated)
	}
}

func TestExtract(t *testing.T) {
	input := "a\n<!-- PROMAN-SYNC-START source=src.md -->\nline1\nline2\n<!-- PROMAN-SYNC-END -->\nb\n"
	block, ok := Extract(input)
	if !ok {
		t.Fatal("expected block")
	}
	if block.Source != "src.md" {
		t.Fatalf("unexpected source: %s", block.Source)
	}
	if block.Content != "line1\nline2" {
		t.Fatalf("unexpected content: %q", block.Content)
	}
}
