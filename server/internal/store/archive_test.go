package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tempStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close(); _ = os.RemoveAll(dir) })
	return st
}

func TestSearchStarExport(t *testing.T) {
	st := tempStore(t)
	book := &Book{BookID: "b1", Title: "人间草木", Author: "汪曾祺", InNotebooks: true, Sort: 10}
	if err := st.UpsertNotebook(book); err != nil {
		t.Fatal(err)
	}
	err := st.ReplaceNotes("b1",
		[]Chapter{{BookID: "b1", ChapterUID: 1, Title: "昆明的雨", ChapterIdx: 1}},
		[]Highlight{{BookmarkID: "h1", BookID: "b1", ChapterUID: 1, MarkText: "雨季的果子是青的", CreateTime: 100}},
		[]Review{{ReviewID: "r1", BookID: "b1", ChapterUID: 1, ChapterName: "昆明的雨", Content: "想吃杨梅", Abstract: "雨季的果子是青的", CreateTime: 90}},
	)
	if err != nil {
		t.Fatal(err)
	}

	hits, err := st.SearchNotes("杨梅", "all", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Kind != "review" {
		t.Fatalf("search review: %+v", hits)
	}
	hits, err = st.SearchNotes("青的", "highlight", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].BookmarkID != "h1" {
		t.Fatalf("search highlight: %+v", hits)
	}

	if err := st.SetHighlightStarred("h1", true); err != nil {
		t.Fatal(err)
	}
	starred, err := st.ListStarredHighlights(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(starred) != 1 || !starred[0].Starred {
		t.Fatalf("starred: %+v", starred)
	}
	hls, err := st.ListHighlights("b1")
	if err != nil {
		t.Fatal(err)
	}
	if len(hls) != 1 || !hls[0].Starred {
		t.Fatalf("list starred flag: %+v", hls)
	}

	pack, err := st.LoadExportBook("b1")
	if err != nil || pack == nil {
		t.Fatalf("export pack %v %v", pack, err)
	}
	md := MarkdownForBook(pack)
	if !strings.Contains(md, "人间草木") || !strings.Contains(md, "雨季的果子是青的") || !strings.Contains(md, "想吃杨梅") {
		t.Fatalf("markdown:\n%s", md)
	}
	if name := SafeFilename(`人间/草木:*`); strings.ContainsAny(name, `/*:`) {
		t.Fatalf("unsafe name %q", name)
	}
}

func TestSearchChinesePhraseNotPartialRune(t *testing.T) {
	st := tempStore(t)
	book := &Book{BookID: "b2", Title: "测试", Author: "作者", InNotebooks: true, Sort: 2}
	if err := st.UpsertNotebook(book); err != nil {
		t.Fatal(err)
	}
	err := st.ReplaceNotes("b2",
		[]Chapter{{BookID: "b2", ChapterUID: 1, Title: "一", ChapterIdx: 1}},
		[]Highlight{
			{BookmarkID: "only-me", BookID: "b2", ChapterUID: 1, MarkText: "我一个人走了", CreateTime: 20},
			{BookmarkID: "we", BookID: "b2", ChapterUID: 1, MarkText: "我们明天出发", CreateTime: 30},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	hits, err := st.SearchNotes("我们", "highlight", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].BookmarkID != "we" {
		t.Fatalf("我们 should match phrase only, got %+v", hits)
	}
	hits, err = st.SearchNotes("我", "highlight", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("我 should match both, got %+v", hits)
	}
	if hits[0].BookmarkID != "we" || hits[1].BookmarkID != "only-me" {
		t.Fatalf("want newer highlight first, got %s then %s", hits[0].BookmarkID, hits[1].BookmarkID)
	}
}

func TestDailyPicksPersistAcrossOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "picks.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	book := &Book{BookID: "b1", Title: "书", Author: "作者", InNotebooks: true, Sort: 1}
	if err := st.UpsertNotebook(book); err != nil {
		t.Fatal(err)
	}
	err = st.ReplaceNotes("b1",
		[]Chapter{{BookID: "b1", ChapterUID: 1, Title: "章", ChapterIdx: 1}},
		[]Highlight{
			{BookmarkID: "h1", BookID: "b1", ChapterUID: 1, MarkText: "一句", CreateTime: 1},
			{BookmarkID: "h2", BookID: "b1", ChapterUID: 1, MarkText: "二句", CreateTime: 2},
		},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SaveDailyPicks("2026-08-25", []RandomHighlight{
		{Highlight: Highlight{BookmarkID: "h2"}},
		{Highlight: Highlight{BookmarkID: "h1"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	items, found, err := st.LoadDailyPicks("2026-08-25")
	if err != nil || !found {
		t.Fatalf("load after reopen: found=%v err=%v", found, err)
	}
	if len(items) != 2 || items[0].BookmarkID != "h2" || items[1].BookmarkID != "h1" {
		t.Fatalf("order: %+v", items)
	}
	if items[0].Title != "书" || items[0].MarkText != "二句" {
		t.Fatalf("joined fields: %+v", items[0])
	}
	_, found, err = st.LoadDailyPicks("2026-08-24")
	if err != nil || found {
		t.Fatalf("other day should miss, found=%v err=%v", found, err)
	}
}
