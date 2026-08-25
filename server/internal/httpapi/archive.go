package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/jachin/weread-helper/internal/store"
)

func (s *Server) searchNotes(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入要检索的字词"})
		return
	}
	kind := c.DefaultQuery("kind", "all")
	limit := queryInt(c, "limit", 40)
	hits, err := s.store.SearchNotes(q, kind, limit)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"query": q, "kind": kind, "items": searchHitsJSON(hits)})
}

func searchHitsJSON(hits []store.NoteSearchHit) []gin.H {
	out := make([]gin.H, 0, len(hits))
	for _, h := range hits {
		out = append(out, gin.H{
			"kind":         h.Kind,
			"bookmarkId":   h.BookmarkID,
			"reviewId":     h.ReviewID,
			"bookId":       h.BookID,
			"title":        h.Title,
			"author":       h.Author,
			"cover":        h.Cover,
			"chapterUid":   h.ChapterUID,
			"chapterTitle": h.ChapterTitle,
			"markText":     h.MarkText,
			"content":      h.Content,
			"abstract":     h.Abstract,
			"createTime":   h.CreateTime,
			"starred":      h.Starred,
		})
	}
	return out
}

func (s *Server) starredHighlights(c *gin.Context) {
	limit := queryInt(c, "limit", 80)
	items, err := s.store.ListStarredHighlights(limit)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": highlightItemsJSON(items)})
}

type starBody struct {
	BookmarkID string `json:"bookmarkId"`
	Starred    *bool  `json:"starred"`
}

func (s *Server) setHighlightStar(c *gin.Context) {
	var body starBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式不对"})
		return
	}
	if strings.TrimSpace(body.BookmarkID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少划线编号"})
		return
	}
	starred := true
	if body.Starred != nil {
		starred = *body.Starred
	}
	if err := s.store.SetHighlightStarred(body.BookmarkID, starred); err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"bookmarkId": body.BookmarkID, "starred": starred})
}

func (s *Server) exportBook(c *gin.Context) {
	bookID := c.Param("bookId")
	pack, err := s.store.LoadExportBook(bookID)
	if err != nil {
		writeErr(c, err)
		return
	}
	if pack == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "本地无此书，请先同步"})
		return
	}
	s.writeExport(c, []*store.ExportBook{pack}, store.SafeFilename(pack.Book.Title))
}

func (s *Server) exportAll(c *gin.Context) {
	packs, err := s.store.LoadExportAll()
	if err != nil {
		writeErr(c, err)
		return
	}
	s.writeExport(c, packs, "纸间笔记-全部")
}

func attachmentDisposition(filename string) string {
	ascii := asciiDownloadName(filename)
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, ascii, url.PathEscape(filename))
}

func asciiDownloadName(filename string) string {
	ext := ""
	if i := strings.LastIndex(filename, "."); i >= 0 && i > len(filename)-6 {
		ext = filename[i:]
		filename = filename[:i]
	}
	var b strings.Builder
	for _, r := range filename {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case unicode.IsSpace(r):
			b.WriteByte('_')
		}
	}
	base := strings.Trim(b.String(), "_")
	if base == "" {
		base = "notes"
	}
	return base + ext
}

func (s *Server) writeExport(c *gin.Context, packs []*store.ExportBook, base string) {
	format := strings.ToLower(strings.TrimSpace(c.DefaultQuery("format", "md")))
	if format == "json" {
		payload := gin.H{
			"exportedAt": time.Now().Unix(),
			"books":      exportBooksJSON(packs),
		}
		raw, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			writeErr(c, err)
			return
		}
		filename := base + ".json"
		c.Header("Content-Disposition", attachmentDisposition(filename))
		c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
		return
	}
	var md string
	if len(packs) == 1 {
		md = store.MarkdownForBook(packs[0])
	} else {
		md = store.MarkdownForBooks(packs)
	}
	filename := base + ".md"
	c.Header("Content-Disposition", attachmentDisposition(filename))
	// UTF-8 BOM so Windows 记事本按 UTF-8 打开，书名不会被当成系统 ANSI。
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", append([]byte{0xEF, 0xBB, 0xBF}, md...))
}

func exportBooksJSON(packs []*store.ExportBook) []gin.H {
	out := make([]gin.H, 0, len(packs))
	for _, pack := range packs {
		hls := make([]gin.H, 0, len(pack.Highlights))
		for _, h := range pack.Highlights {
			hls = append(hls, gin.H{
				"bookmarkId": h.BookmarkID,
				"chapterUid": h.ChapterUID,
				"markText":   h.MarkText,
				"createTime": h.CreateTime,
				"starred":    h.Starred,
			})
		}
		revs := make([]gin.H, 0, len(pack.Reviews))
		for _, r := range pack.Reviews {
			revs = append(revs, gin.H{
				"reviewId":    r.ReviewID,
				"chapterUid":  r.ChapterUID,
				"chapterName": r.ChapterName,
				"content":     r.Content,
				"abstract":    r.Abstract,
				"createTime":  r.CreateTime,
			})
		}
		chs := make([]gin.H, 0, len(pack.Chapters))
		for _, ch := range pack.Chapters {
			chs = append(chs, gin.H{
				"chapterUid": ch.ChapterUID,
				"title":      ch.Title,
				"chapterIdx": ch.ChapterIdx,
			})
		}
		out = append(out, gin.H{
			"bookId":     pack.Book.BookID,
			"title":      pack.Book.Title,
			"author":     pack.Book.Author,
			"cover":      pack.Book.Cover,
			"chapters":   chs,
			"highlights": hls,
			"reviews":    revs,
		})
	}
	return out
}
