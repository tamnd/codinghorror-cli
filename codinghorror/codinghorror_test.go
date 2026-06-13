package codinghorror_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/codinghorror-cli/codinghorror"
)

// rssXML wraps items in a minimal valid RSS 2.0 feed.
func rssXML(items string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:dc="http://purl.org/dc/elements/1.1/">
  <channel>
` + items + `
  </channel>
</rss>`
}

func singleItem(title, link, pubDate, category, description string) string {
	return `<item>
  <title>` + title + `</title>
  <link>` + link + `</link>
  <pubDate>` + pubDate + `</pubDate>
  <category><![CDATA[` + category + `]]></category>
  <description><![CDATA[` + description + `]]></description>
</item>`
}

func newTestClient(ts *httptest.Server) *codinghorror.Client {
	cfg := codinghorror.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	return codinghorror.NewClient(cfg)
}

func TestLatestParsesTitle(t *testing.T) {
	xml := rssXML(singleItem(
		"Code Tells You How, Comments Tell You Why",
		"https://blog.codinghorror.com/code-tells-you-how/",
		"Mon, 15 Jan 2024 12:00:00 +0000",
		"programming",
		"<p>A short summary about comments.</p>",
	))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(xml))
	}))
	defer ts.Close()

	posts, err := newTestClient(ts).Latest(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 {
		t.Fatalf("got %d posts, want 1", len(posts))
	}
	if posts[0].Title != "Code Tells You How, Comments Tell You Why" {
		t.Errorf("Title = %q", posts[0].Title)
	}
}

func TestLatestParsesURL(t *testing.T) {
	wantURL := "https://blog.codinghorror.com/the-best-code-is-no-code-at-all/"
	xml := rssXML(singleItem(
		"The Best Code Is No Code At All",
		wantURL,
		"Fri, 12 Jan 2024 15:30:00 +0000",
		"programming",
		"<p>Summary here.</p>",
	))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(xml))
	}))
	defer ts.Close()

	posts, err := newTestClient(ts).Latest(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if posts[0].URL != wantURL {
		t.Errorf("URL = %q, want %q", posts[0].URL, wantURL)
	}
}

func TestLatestParsesDate(t *testing.T) {
	xml := rssXML(singleItem(
		"Programming Tip",
		"https://blog.codinghorror.com/tip/",
		"Thu, 07 Mar 2024 18:00:00 GMT",
		"",
		"<p>Details.</p>",
	))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(xml))
	}))
	defer ts.Close()

	posts, err := newTestClient(ts).Latest(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if posts[0].Published != "2024-03-07" {
		t.Errorf("Published = %q, want %q", posts[0].Published, "2024-03-07")
	}
}

func TestLatestStripsSummaryHTML(t *testing.T) {
	xml := rssXML(singleItem(
		"HTML Stripping",
		"https://blog.codinghorror.com/html/",
		"Sat, 20 Jan 2024 10:00:00 +0000",
		"",
		"<p>This is the <b>summary</b> text.</p>",
	))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(xml))
	}))
	defer ts.Close()

	posts, err := newTestClient(ts).Latest(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(posts[0].Summary, "<") || strings.Contains(posts[0].Summary, ">") {
		t.Errorf("Summary contains HTML tags: %q", posts[0].Summary)
	}
	if !strings.Contains(posts[0].Summary, "summary") {
		t.Errorf("Summary text missing: %q", posts[0].Summary)
	}
}

func TestLatestTruncatesSummary(t *testing.T) {
	long := strings.Repeat("x", 300)
	xml := rssXML(singleItem(
		"Long Post",
		"https://blog.codinghorror.com/long/",
		"Mon, 01 Jan 2024 00:00:00 +0000",
		"",
		"<p>"+long+"</p>",
	))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(xml))
	}))
	defer ts.Close()

	posts, err := newTestClient(ts).Latest(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	runes := []rune(posts[0].Summary)
	if len(runes) > 150 {
		t.Errorf("Summary too long: %d runes", len(runes))
	}
	if !strings.HasSuffix(posts[0].Summary, "…") {
		t.Errorf("Summary missing ellipsis: %q", posts[0].Summary)
	}
}

func TestLatestRankOrder(t *testing.T) {
	items := singleItem("A", "https://blog.codinghorror.com/a/", "Mon, 01 Jan 2024 00:00:00 +0000", "", "") +
		singleItem("B", "https://blog.codinghorror.com/b/", "Tue, 02 Jan 2024 00:00:00 +0000", "", "") +
		singleItem("C", "https://blog.codinghorror.com/c/", "Wed, 03 Jan 2024 00:00:00 +0000", "", "")
	xml := rssXML(items)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(xml))
	}))
	defer ts.Close()

	posts, err := newTestClient(ts).Latest(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 3 {
		t.Fatalf("got %d posts, want 3", len(posts))
	}
	for i, p := range posts {
		if p.Rank != i+1 {
			t.Errorf("posts[%d].Rank = %d, want %d", i, p.Rank, i+1)
		}
	}
}

func TestLatestLimit(t *testing.T) {
	items := ""
	for i := 0; i < 5; i++ {
		items += singleItem("T", "https://blog.codinghorror.com/t/", "Mon, 01 Jan 2024 00:00:00 +0000", "", "")
	}
	xml := rssXML(items)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(xml))
	}))
	defer ts.Close()

	posts, err := newTestClient(ts).Latest(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 2 {
		t.Errorf("got %d posts with limit=2, want 2", len(posts))
	}
}

func TestSearchFiltersbyTitle(t *testing.T) {
	items := singleItem("Programming Tips", "https://blog.codinghorror.com/prog/", "Mon, 01 Jan 2024 00:00:00 +0000", "", "All about code.") +
		singleItem("Coffee and Gaming", "https://blog.codinghorror.com/coffee/", "Tue, 02 Jan 2024 00:00:00 +0000", "", "A gaming post.")
	xml := rssXML(items)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(xml))
	}))
	defer ts.Close()

	posts, err := newTestClient(ts).Search(context.Background(), "programming", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 {
		t.Fatalf("got %d posts, want 1", len(posts))
	}
	if posts[0].Title != "Programming Tips" {
		t.Errorf("Title = %q", posts[0].Title)
	}
}

func TestSearchFiltersBySummary(t *testing.T) {
	items := singleItem("Random Post", "https://blog.codinghorror.com/rand/", "Mon, 01 Jan 2024 00:00:00 +0000", "", "This talks about keyboards.") +
		singleItem("Another Post", "https://blog.codinghorror.com/other/", "Tue, 02 Jan 2024 00:00:00 +0000", "", "Nothing relevant here.")
	xml := rssXML(items)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(xml))
	}))
	defer ts.Close()

	posts, err := newTestClient(ts).Search(context.Background(), "keyboards", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 1 {
		t.Fatalf("got %d posts, want 1", len(posts))
	}
}

func TestSearchReturnsEmpty(t *testing.T) {
	items := singleItem("Programming", "https://blog.codinghorror.com/prog/", "Mon, 01 Jan 2024 00:00:00 +0000", "", "Code is great.")
	xml := rssXML(items)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(xml))
	}))
	defer ts.Close()

	posts, err := newTestClient(ts).Search(context.Background(), "zyxwvutsrq", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts) != 0 {
		t.Errorf("got %d posts, want 0", len(posts))
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	items := singleItem("Retry Test", "https://blog.codinghorror.com/retry/", "Mon, 01 Jan 2024 00:00:00 +0000", "", "")
	xml := rssXML(items)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(xml))
	}))
	defer ts.Close()

	cfg := codinghorror.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 5
	c := codinghorror.NewClient(cfg)

	start := time.Now()
	_, err := c.Latest(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestGetUserAgent(t *testing.T) {
	var gotUA string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(rssXML("")))
	}))
	defer ts.Close()

	cfg := codinghorror.DefaultConfig()
	cfg.BaseURL = ts.URL
	cfg.Rate = 0
	c := codinghorror.NewClient(cfg)
	_, _ = c.Latest(context.Background(), 0)

	if gotUA == "" {
		t.Error("request carried no User-Agent")
	}
}
