package weread

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Client struct {
	mu           sync.Mutex
	http         *http.Client
	gatewayURL   string
	apiKey       string
	skillVersion string
}

func New(gatewayURL, apiKey, skillVersion string) *Client {
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		gatewayURL:   gatewayURL,
		apiKey:       apiKey,
		skillVersion: skillVersion,
	}
}

func (c *Client) Update(gatewayURL, apiKey, skillVersion string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gatewayURL = gatewayURL
	c.apiKey = apiKey
	c.skillVersion = skillVersion
}

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Upgrade any    `json:"upgrade,omitempty"`
}

func (e *APIError) Error() string {
	return e.Message
}

func (c *Client) Call(apiName string, params map[string]any) (map[string]any, error) {
	c.mu.Lock()
	gw, key, ver := c.gatewayURL, c.apiKey, c.skillVersion
	c.mu.Unlock()

	payload := map[string]any{
		"api_name":      apiName,
		"skill_version": ver,
	}
	for k, v := range params {
		payload[k] = v
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, gw, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{Code: resp.StatusCode, Message: fmt.Sprintf("gateway HTTP %d: %s", resp.StatusCode, truncate(string(raw), 300))}
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("invalid json: %w", err)
	}

	if upgrade, ok := data["upgrade_info"]; ok && upgrade != nil {
		return nil, &APIError{Code: 426, Message: "微信读书 API 需要升级 skill_version", Upgrade: upgrade}
	}

	if code, ok := asInt(data["errcode"]); ok && code != 0 {
		msg, _ := data["errmsg"].(string)
		if msg == "" {
			msg = "微信读书 API 调用失败"
		}
		return nil, &APIError{Code: code, Message: msg}
	}

	return data, nil
}

func (c *Client) Notebooks(count int, lastSort int64) (map[string]any, error) {
	params := map[string]any{"count": count}
	if lastSort > 0 {
		params["lastSort"] = lastSort
	}
	return c.Call("/user/notebooks", params)
}

func (c *Client) Highlights(bookID string) (map[string]any, error) {
	return c.Call("/book/bookmarklist", map[string]any{"bookId": bookID})
}

func (c *Client) MyReviews(bookID string, count int, synckey int64) (map[string]any, error) {
	params := map[string]any{
		"bookid": bookID,
		"count":  count,
	}
	if synckey > 0 {
		params["synckey"] = synckey
	}
	return c.Call("/review/list/mine", params)
}

func (c *Client) ReadStats(mode string, baseTime int64) (map[string]any, error) {
	params := map[string]any{"mode": mode}
	if baseTime > 0 {
		params["baseTime"] = baseTime
	}
	return c.Call("/readdata/detail", params)
}

func (c *Client) BookInfo(bookID string) (map[string]any, error) {
	return c.Call("/book/info", map[string]any{"bookId": bookID})
}

func (c *Client) Chapters(bookID string) (map[string]any, error) {
	return c.Call("/book/chapterinfo", map[string]any{"bookId": bookID})
}

func (c *Client) Progress(bookID string) (map[string]any, error) {
	return c.Call("/book/getprogress", map[string]any{"bookId": bookID})
}

func (c *Client) Shelf() (map[string]any, error) {
	return c.Call("/shelf/sync", nil)
}

func asInt(v any) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case json.Number:
		i, err := n.Int64()
		return int(i), err == nil
	default:
		return 0, false
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
