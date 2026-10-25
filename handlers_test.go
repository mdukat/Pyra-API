package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	dict := testDictionary(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/", infoHandler(dict))
	mux.HandleFunc("/healthz", healthHandler(dict))
	mux.HandleFunc("/translate", translateHandler(dict))
	mux.HandleFunc("/dictionary", dictionaryHandler(dict))
	srv := httptest.NewServer(withCORS(mux))
	t.Cleanup(srv.Close)
	return srv
}

func decodeJSON(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
}

func TestHTTPTranslateQuery(t *testing.T) {
	srv := newTestServer(t)
	text := url.QueryEscape("Ziemniaki są w kuchni.")
	resp, err := http.Get(srv.URL + "/translate?text=" + text)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("content type = %q", ct)
	}
	var result TranslationResult
	decodeJSON(t, resp, &result)
	if result.Translation != "Pyry są wew kuchni." {
		t.Errorf("translation = %q", result.Translation)
	}
	if result.Input != "Ziemniaki są w kuchni." {
		t.Errorf("input = %q", result.Input)
	}
}

func TestHTTPTranslateJSONBody(t *testing.T) {
	srv := newTestServer(t)
	body := `{"text":"Mam ziemniaki.","preserve_case":true,"max_alternatives":2}`
	resp, err := http.Post(srv.URL+"/translate", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var result TranslationResult
	decodeJSON(t, resp, &result)
	if result.Translation != "Mam pyry." {
		t.Errorf("translation = %q", result.Translation)
	}
}

func TestHTTPTranslateMissingText(t *testing.T) {
	srv := newTestServer(t)
	resp, err := http.Get(srv.URL + "/translate")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var errResp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	decodeJSON(t, resp, &errResp)
	if errResp.Error.Code != "missing_text" {
		t.Errorf("error code = %q", errResp.Error.Code)
	}
}

func TestHTTPHealthAndDictionary(t *testing.T) {
	srv := newTestServer(t)

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	var health struct {
		Status string `json:"status"`
	}
	decodeJSON(t, resp, &health)
	if health.Status != "ok" {
		t.Errorf("health = %+v", health)
	}

	resp, err = http.Get(srv.URL + "/dictionary?q=pyra&direction=dialect&limit=5")
	if err != nil {
		t.Fatal(err)
	}
	var dictResp struct {
		Count   int `json:"count"`
		Entries []struct {
			Dialect  string `json:"dialect"`
			Standard string `json:"standard"`
		} `json:"entries"`
	}
	decodeJSON(t, resp, &dictResp)
	if dictResp.Count == 0 {
		t.Fatalf("expected dictionary results")
	}
	foundPyra := false
	for _, e := range dictResp.Entries {
		if e.Dialect == "pyra" {
			foundPyra = true
		}
	}
	if !foundPyra {
		t.Errorf("expected pyra in dictionary results: %+v", dictResp.Entries)
	}
}

func TestHTTPOptionsCORS(t *testing.T) {
	srv := newTestServer(t)
	req, err := http.NewRequest(http.MethodOptions, srv.URL+"/translate", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d", resp.StatusCode)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("missing CORS header")
	}
}
