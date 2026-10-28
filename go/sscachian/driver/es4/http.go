package es4

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	es4lib "github.com/b4moss/es4/packages/go/pkg/es4"
)

// HTTPOptions configures NewHTTP (Es4 Server adapter).
type HTTPOptions struct {
	// BaseURL is the Es4 Server root (e.g. http://127.0.0.1:8080). Required.
	BaseURL string
	// HTTPClient overrides the default client. Optional.
	HTTPClient *http.Client
}

// NewHTTP creates a Layer that talks to an Es4 HTTP Server.
// Key mapping follows es4 decision A: /v1/keys/a:b:c.
// Scan uses GET /v1/scan?prefix= (es4 v0.10+ Server).
func NewHTTP(ctx context.Context, opts HTTPOptions) (*Store, error) {
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("sscachian/es4: BaseURL is required")
	}
	base := strings.TrimRight(opts.BaseURL, "/")
	client := opts.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	b := &httpBackend{base: base, client: client}
	if err := b.pingReady(ctx); err != nil {
		return nil, err
	}
	return &Store{backend: b, ownsDB: true}, nil
}

type httpBackend struct {
	base   string
	client *http.Client
}

func (b *httpBackend) pingReady(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.base+"/readyz", nil)
	if err != nil {
		return err
	}
	res, err := b.client.Do(req)
	if err != nil {
		return fmt.Errorf("sscachian/es4: connect: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusServiceUnavailable {
		return fmt.Errorf("sscachian/es4: server not ready (503)")
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("sscachian/es4: readyz status %d", res.StatusCode)
	}
	return nil
}

func (b *httpBackend) keyURL(key string) string {
	// Decision A: single path segment with ':' kept as-is (escaped for URL).
	return b.base + "/v1/keys/" + url.PathEscape(key)
}

func (b *httpBackend) Set(ctx context.Context, key string, value json.RawMessage, expiresAt *time.Time) error {
	u := b.keyURL(key)
	if expiresAt != nil {
		u += "?expires_at=" + url.QueryEscape(expiresAt.UTC().Format(time.RFC3339))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(value))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return mapHTTPStatus(res)
}

func (b *httpBackend) Get(ctx context.Context, key string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.keyURL(key), nil)
	if err != nil {
		return nil, err
	}
	res, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, es4lib.ErrNotFound
	}
	if err := mapHTTPStatus(res); err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func (b *httpBackend) Delete(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, b.keyURL(key), nil)
	if err != nil {
		return err
	}
	res, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return es4lib.ErrNotFound
	}
	return mapHTTPStatus(res)
}

func (b *httpBackend) Scan(ctx context.Context, prefix string) ([]string, error) {
	u := b.base + "/v1/scan?prefix=" + url.QueryEscape(prefix)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if err := mapHTTPStatus(res); err != nil {
		return nil, err
	}
	var keys []string
	if err := json.NewDecoder(res.Body).Decode(&keys); err != nil {
		return nil, fmt.Errorf("sscachian/es4: decode scan: %w", err)
	}
	if keys == nil {
		keys = []string{}
	}
	return keys, nil
}

func (b *httpBackend) BeginTx(ctx context.Context) (Tx, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.base+"/v1/tx", nil)
	if err != nil {
		return nil, err
	}
	res, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if err := mapHTTPStatus(res); err != nil {
		return nil, err
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.ID == "" {
		return nil, fmt.Errorf("sscachian/es4: empty tx id")
	}
	return &httpTx{b: b, id: out.ID}, nil
}

func (b *httpBackend) Close() error {
	return nil
}

type httpTx struct {
	b    *httpBackend
	id   string
	mu   sync.Mutex
	done bool
}

func (t *httpTx) txKeyURL(key string) string {
	return t.b.base + "/v1/tx/" + url.PathEscape(t.id) + "/keys/" + url.PathEscape(key)
}

func (t *httpTx) Set(ctx context.Context, key string, value json.RawMessage, expiresAt *time.Time) error {
	u := t.txKeyURL(key)
	if expiresAt != nil {
		u += "?expires_at=" + url.QueryEscape(expiresAt.UTC().Format(time.RFC3339))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(value))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := t.b.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return mapHTTPStatus(res)
}

func (t *httpTx) Get(ctx context.Context, key string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.txKeyURL(key), nil)
	if err != nil {
		return nil, err
	}
	res, err := t.b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, es4lib.ErrNotFound
	}
	if err := mapHTTPStatus(res); err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(body), nil
}

func (t *httpTx) Delete(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, t.txKeyURL(key), nil)
	if err != nil {
		return err
	}
	res, err := t.b.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return es4lib.ErrNotFound
	}
	return mapHTTPStatus(res)
}

func (t *httpTx) Scan(ctx context.Context, prefix string) ([]string, error) {
	// Provisional until Es4 Server exposes Tx Scan; non-Tx /v1/scan is used.
	u := t.b.base + "/v1/scan?prefix=" + url.QueryEscape(prefix)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := t.b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if err := mapHTTPStatus(res); err != nil {
		return nil, err
	}
	var keys []string
	if err := json.NewDecoder(res.Body).Decode(&keys); err != nil {
		return nil, fmt.Errorf("sscachian/es4: decode tx scan: %w", err)
	}
	if keys == nil {
		keys = []string{}
	}
	return keys, nil
}

func (t *httpTx) Exists(ctx context.Context, key string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, t.txKeyURL(key), nil)
	if err != nil {
		return false, err
	}
	res, err := t.b.client.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if err := mapHTTPStatus(res); err != nil {
		return false, err
	}
	return true, nil
}

func (t *httpTx) Clear(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.b.base+"/v1/tx/"+url.PathEscape(t.id)+"/clear", nil)
	if err != nil {
		return err
	}
	res, err := t.b.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return mapHTTPStatus(res)
}

func (t *httpTx) Commit() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done {
		return es4lib.ErrTxDone
	}
	req, err := http.NewRequest(http.MethodPost, t.b.base+"/v1/tx/"+url.PathEscape(t.id)+"/commit", nil)
	if err != nil {
		return err
	}
	res, err := t.b.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err := mapHTTPStatus(res); err != nil {
		return err
	}
	t.done = true
	return nil
}

func (t *httpTx) Rollback() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.done {
		return es4lib.ErrTxDone
	}
	req, err := http.NewRequest(http.MethodPost, t.b.base+"/v1/tx/"+url.PathEscape(t.id)+"/rollback", nil)
	if err != nil {
		return err
	}
	res, err := t.b.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err := mapHTTPStatus(res); err != nil {
		return err
	}
	t.done = true
	return nil
}

func mapHTTPStatus(res *http.Response) error {
	switch res.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return es4lib.ErrNotFound
	case http.StatusBadRequest:
		code, msg := readErrBody(res.Body)
		if code == "invalid_key" {
			return fmt.Errorf("%w: %s", es4lib.ErrInvalidKey, msg)
		}
		if code == "invalid_value" {
			return fmt.Errorf("%w: %s", es4lib.ErrInvalidValue, msg)
		}
		return fmt.Errorf("sscachian/es4: bad request: %s", msg)
	case http.StatusServiceUnavailable:
		return fmt.Errorf("sscachian/es4: server not ready (503)")
	case http.StatusConflict:
		return es4lib.ErrTxDone
	default:
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("sscachian/es4: http %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
}

func readErrBody(r io.Reader) (code, message string) {
	var e struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	_ = json.NewDecoder(r).Decode(&e)
	return e.Error.Code, e.Error.Message
}
