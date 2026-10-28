package es4_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/b4moss/ss-cachian/go/sscachian"
	es4driver "github.com/b4moss/ss-cachian/go/sscachian/driver/es4"
)

// mockEs4Server approximates Es4 HTTP Server for adapter tests.
func mockEs4Server(t *testing.T, ready bool) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	store := map[string]json.RawMessage{}
	type txState struct {
		overlay map[string]*json.RawMessage
		done    bool
	}
	txs := map[string]*txState{}
	txSeq := 0

	mux := http.NewServeMux()
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !ready {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("PUT /v1/keys/{key}", func(w http.ResponseWriter, r *http.Request) {
		if !ready {
			writeJSONErr(w, http.StatusServiceUnavailable, "not_ready", "restore not ready")
			return
		}
		key := r.PathValue("key")
		body, err := readBody(r)
		if err != nil {
			writeJSONErr(w, http.StatusBadRequest, "invalid_value", err.Error())
			return
		}
		mu.Lock()
		store[key] = body
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /v1/keys/{key}", func(w http.ResponseWriter, r *http.Request) {
		if !ready {
			writeJSONErr(w, http.StatusServiceUnavailable, "not_ready", "restore not ready")
			return
		}
		key := r.PathValue("key")
		mu.Lock()
		v, ok := store[key]
		mu.Unlock()
		if !ok {
			writeJSONErr(w, http.StatusNotFound, "not_found", "missing")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(v)
	})
	mux.HandleFunc("DELETE /v1/keys/{key}", func(w http.ResponseWriter, r *http.Request) {
		if !ready {
			writeJSONErr(w, http.StatusServiceUnavailable, "not_ready", "restore not ready")
			return
		}
		key := r.PathValue("key")
		mu.Lock()
		_, ok := store[key]
		if ok {
			delete(store, key)
		}
		mu.Unlock()
		if !ok {
			writeJSONErr(w, http.StatusNotFound, "not_found", "missing")
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /v1/scan", func(w http.ResponseWriter, r *http.Request) {
		if !ready {
			writeJSONErr(w, http.StatusServiceUnavailable, "not_ready", "restore not ready")
			return
		}
		prefix := r.URL.Query().Get("prefix")
		mu.Lock()
		var keys []string
		for k := range store {
			if strings.HasPrefix(k, prefix) {
				keys = append(keys, k)
			}
		}
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(keys)
	})
	mux.HandleFunc("POST /v1/tx", func(w http.ResponseWriter, r *http.Request) {
		if !ready {
			writeJSONErr(w, http.StatusServiceUnavailable, "not_ready", "restore not ready")
			return
		}
		mu.Lock()
		txSeq++
		id := "tx-" + itoa(txSeq)
		txs[id] = &txState{overlay: map[string]*json.RawMessage{}}
		mu.Unlock()
		writeJSON(w, map[string]string{"id": id})
	})
	mux.HandleFunc("PUT /v1/tx/{id}/keys/{key}", func(w http.ResponseWriter, r *http.Request) {
		body, err := readBody(r)
		if err != nil {
			writeJSONErr(w, http.StatusBadRequest, "invalid_value", err.Error())
			return
		}
		mu.Lock()
		tx := txs[r.PathValue("id")]
		if tx == nil || tx.done {
			mu.Unlock()
			writeJSONErr(w, http.StatusNotFound, "not_found", "tx")
			return
		}
		cp := append(json.RawMessage(nil), body...)
		tx.overlay[r.PathValue("key")] = &cp
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /v1/tx/{id}/keys/{key}", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		tx := txs[r.PathValue("id")]
		key := r.PathValue("key")
		var v json.RawMessage
		var ok bool
		if tx != nil && !tx.done {
			if ov, hit := tx.overlay[key]; hit {
				if ov != nil {
					v, ok = *ov, true
				}
			} else if sv, hit := store[key]; hit {
				v, ok = sv, true
			}
		}
		mu.Unlock()
		if !ok {
			writeJSONErr(w, http.StatusNotFound, "not_found", "missing")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(v)
	})
	mux.HandleFunc("POST /v1/tx/{id}/commit", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		tx := txs[r.PathValue("id")]
		if tx == nil || tx.done {
			mu.Unlock()
			writeJSONErr(w, http.StatusNotFound, "not_found", "tx")
			return
		}
		for k, v := range tx.overlay {
			if v == nil {
				delete(store, k)
			} else {
				store[k] = append(json.RawMessage(nil), (*v)...)
			}
		}
		tx.done = true
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /v1/tx/{id}/rollback", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		tx := txs[r.PathValue("id")]
		if tx != nil {
			tx.done = true
		}
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestHTTP_SetGetPurgePrefix(t *testing.T) {
	t.Parallel()
	srv := mockEs4Server(t, true)
	ctx := context.Background()
	s, err := es4driver.NewHTTP(ctx, es4driver.HTTPOptions{BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if err := s.Set(ctx, "a:b:1", sscachian.Entry{Value: "v"}, 0); err != nil {
		t.Fatal(err)
	}
	e, ok, err := s.Get(ctx, "a:b:1")
	if err != nil || !ok || e.Value != "v" {
		t.Fatalf("ok=%v val=%v err=%v", ok, e.Value, err)
	}
	_ = s.Set(ctx, "a:b:__version__", sscachian.Entry{Value: 1}, 0)
	_ = s.Set(ctx, "other:1", sscachian.Entry{Value: 1}, 0)
	if err := s.PurgePrefix(ctx, "a:b"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Get(ctx, "a:b:1"); ok {
		t.Fatal("purged")
	}
	if _, ok, _ := s.Get(ctx, "other:1"); !ok {
		t.Fatal("other should remain")
	}
}

func TestHTTP_NotReadyAndConnectFail(t *testing.T) {
	t.Parallel()
	srv := mockEs4Server(t, false)
	_, err := es4driver.NewHTTP(context.Background(), es4driver.HTTPOptions{BaseURL: srv.URL})
	if err == nil {
		t.Fatal("expected not ready error")
	}
	_, err = es4driver.NewHTTP(context.Background(), es4driver.HTTPOptions{BaseURL: "http://127.0.0.1:1"})
	if err == nil {
		t.Fatal("expected connect error")
	}
}

func TestHTTP_CloseThenError(t *testing.T) {
	t.Parallel()
	srv := mockEs4Server(t, true)
	s, err := es4driver.NewHTTP(context.Background(), es4driver.HTTPOptions{BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	_ = s.Close()
	if err := s.Set(context.Background(), "k", sscachian.Entry{Value: 1}, 0); err == nil {
		t.Fatal("expected closed error")
	}
}

func TestHTTP_Incr(t *testing.T) {
	t.Parallel()
	srv := mockEs4Server(t, true)
	s, err := es4driver.NewHTTP(context.Background(), es4driver.HTTPOptions{BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	n, err := s.Incr(context.Background(), "c")
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	n, err = s.Incr(context.Background(), "c")
	if err != nil || n != 2 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func writeJSONErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": msg},
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func readBody(r *http.Request) (json.RawMessage, error) {
	var raw json.RawMessage
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
