package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jachin/weread-helper/internal/store"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func testAPI(t *testing.T) (*gin.Engine, *store.Store) {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	s := New(st, nil, nil, nil)
	r := gin.New()
	s.Register(r)
	return r, st
}

func seedSixHighlights(t *testing.T, st *store.Store, n int) {
	t.Helper()
	book := &store.Book{BookID: "b1", Title: "书", Author: "作者", InNotebooks: true, Sort: 1}
	if err := st.UpsertNotebook(book); err != nil {
		t.Fatal(err)
	}
	hls := make([]store.Highlight, 0, n)
	for i := 0; i < n; i++ {
		hls = append(hls, store.Highlight{
			BookmarkID: "h" + strconv.Itoa(i+1),
			BookID:     "b1",
			ChapterUID: 1,
			MarkText:   "句" + strconv.Itoa(i+1),
			CreateTime: int64(i + 1),
		})
	}
	err := st.ReplaceNotes("b1",
		[]store.Chapter{{BookID: "b1", ChapterUID: 1, Title: "章", ChapterIdx: 1}},
		hls,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
}

func saveFirstNPicks(t *testing.T, st *store.Store, n int) string {
	t.Helper()
	today := time.Now().In(store.Shanghai()).Format("2006-01-02")
	items := make([]store.RandomHighlight, n)
	for i := 0; i < n; i++ {
		items[i] = store.RandomHighlight{Highlight: store.Highlight{BookmarkID: "h" + strconv.Itoa(i+1)}}
	}
	if err := st.SaveDailyPicks(today, items); err != nil {
		t.Fatal(err)
	}
	return today
}

func postRedraw(r *gin.Engine, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/highlights/random/redraw", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

type redrawOut struct {
	Date  string `json:"date"`
	Error string `json:"error"`
	Items []struct {
		BookmarkID string `json:"bookmarkId"`
	} `json:"items"`
}

func parseRedraw(t *testing.T, w *httptest.ResponseRecorder) redrawOut {
	t.Helper()
	var out redrawOut
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("json: %v body=%s", err, w.Body.String())
	}
	return out
}

func TestRedrawReplacesOneSlotAndKeepsOthers(t *testing.T) {
	r, st := testAPI(t)
	seedSixHighlights(t, st, 6)
	today := saveFirstNPicks(t, st, 5)

	w := postRedraw(r, `{"pos":0}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	out := parseRedraw(t, w)
	if out.Date != today || len(out.Items) != 5 {
		t.Fatalf("date/items: %+v", out)
	}
	if out.Items[0].BookmarkID != "h6" {
		t.Fatalf("pos 0 want h6, got %s", out.Items[0].BookmarkID)
	}
	for i := 1; i < 5; i++ {
		want := "h" + strconv.Itoa(i+1)
		if out.Items[i].BookmarkID != want {
			t.Fatalf("pos %d want %s got %s", i, want, out.Items[i].BookmarkID)
		}
	}
	stored, found, err := st.LoadDailyPicks(today)
	if err != nil || !found {
		t.Fatalf("persist found=%v err=%v", found, err)
	}
	if stored[0].BookmarkID != "h6" || stored[1].BookmarkID != "h2" {
		t.Fatalf("stored %+v", stored)
	}
}

func TestRedrawWithoutDailyPicksDoesNotCreate(t *testing.T) {
	r, st := testAPI(t)
	seedSixHighlights(t, st, 6)
	today := time.Now().In(store.Shanghai()).Format("2006-01-02")

	w := postRedraw(r, `{"pos":0}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	_, found, err := st.LoadDailyPicks(today)
	if err != nil || found {
		t.Fatalf("must not create picks, found=%v err=%v", found, err)
	}
}

func TestRedrawInvalidPos(t *testing.T) {
	r, st := testAPI(t)
	seedSixHighlights(t, st, 6)
	saveFirstNPicks(t, st, 5)

	w := postRedraw(r, `{"pos":9}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}

func TestRedrawNoCandidateLeavesPicksUnchanged(t *testing.T) {
	r, st := testAPI(t)
	seedSixHighlights(t, st, 5)
	today := saveFirstNPicks(t, st, 5)

	w := postRedraw(r, `{"pos":2}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	out := parseRedraw(t, w)
	if out.Error != "没有更多可换的划线" {
		t.Fatalf("error %q", out.Error)
	}
	stored, found, err := st.LoadDailyPicks(today)
	if err != nil || !found || len(stored) != 5 {
		t.Fatalf("found=%v len=%d err=%v", found, len(stored), err)
	}
	if stored[2].BookmarkID != "h3" {
		t.Fatalf("slot 2 changed: %+v", stored)
	}
}
