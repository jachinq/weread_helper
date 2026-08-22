package httpapi

import (
	"net/url"
	"testing"
)

func TestAllowedCoverHost(t *testing.T) {
	ok := []string{
		"weread.qq.com",
		"cdn.weread.qq.com",
		"res.weread.qq.com",
		"wfqqreader-1252317822.image.myqcloud.com",
		"mmbiz.qpic.cn",
		"rescdn.qqmail.com",
		"wrco-40036.sh.gfp.tencent-cloud.com",
	}
	for _, h := range ok {
		if !allowedCoverHost(h) {
			t.Fatalf("expected allowed: %s", h)
		}
	}
	deny := []string{"example.com", "qq.com", "evil.myqcloud.com.attacker.com"}
	for _, h := range deny {
		if allowedCoverHost(h) {
			t.Fatalf("expected denied: %s", h)
		}
	}
}

func TestValidateCoverURL(t *testing.T) {
	u, _ := url.Parse("https://cdn.weread.qq.com/cover.jpg")
	if err := validateCoverURL(u); err != nil {
		t.Fatal(err)
	}
	u, _ = url.Parse("http://cdn.weread.qq.com/cover.jpg")
	if err := validateCoverURL(u); err == nil {
		t.Fatal("http should be denied")
	}
	u, _ = url.Parse("https://127.0.0.1/cover.jpg")
	if err := validateCoverURL(u); err == nil {
		t.Fatal("ip host should be denied by allowlist")
	}
}

func TestSniffImageType(t *testing.T) {
	if g := sniffImageType([]byte{0xff, 0xd8, 0xff, 0xe0}); g != "image/jpeg" {
		t.Fatalf("jpeg: %s", g)
	}
}

func TestUnwrapNestedCoverProxy(t *testing.T) {
	inner := "https://cdn.weread.qq.com/cover.jpg"
	u, _ := url.Parse("http://localhost:5173/api/covers?url=" + url.QueryEscape(inner))
	got := unwrapNestedCoverProxy(u)
	if got.String() != inner {
		t.Fatalf("got %s", got)
	}
}
