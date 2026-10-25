package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

const maxRequestBody = 1 << 20 // 1 MiB

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func infoHandler(d *Dictionary) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			writeError(w, http.StatusNotFound, "not_found", "unknown endpoint")
			return
		}
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"name":               "Pyra Translation API",
			"description":        "Translates standard Polish text into the Poznań dialect.",
			"version":            "1.0.0",
			"dictionary_entries": len(d.Entries),
			"endpoints": map[string]string{
				"translate":  "GET/POST /translate?text=...",
				"dictionary": "GET /dictionary?q=...",
				"health":     "GET /healthz",
			},
			"example": "/translate?text=Ziemniaki%20s%C4%85%20w%20kuchni.",
		})
	}
}

func healthHandler(d *Dictionary) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":             "ok",
			"dictionary_entries": len(d.Entries),
		})
	}
}

type translateRequest struct {
	Text               string `json:"text"`
	Q                  string `json:"q"`
	Input              string `json:"input"`
	PreserveCase       *bool  `json:"preserve_case"`
	PreserveCaseCamel  *bool  `json:"preserveCase"`
	MaxAlternatives    *int   `json:"max_alternatives"`
	MaxAlternativesAlt *int   `json:"maxAlternatives"`
}

func translateHandler(d *Dictionary) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodPost:
		default:
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET or POST")
			return
		}
		text, opts, err := parseTranslateRequest(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		if strings.TrimSpace(text) == "" {
			writeError(w, http.StatusBadRequest, "missing_text",
				"provide the text to translate as ?text=..., ?q=..., ?input=... or a JSON body {\"text\":\"...\"}")
			return
		}
		result := d.Translate(text, opts)
		writeJSON(w, http.StatusOK, result)
	}
}

func parseTranslateRequest(r *http.Request) (string, TranslateOptions, error) {
	opts := TranslateOptions{PreserveCase: true, MaxAlternatives: 5}

	var body translateRequest
	if r.Body != nil {
		data, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBody))
		if err != nil {
			return "", opts, err
		}
		trimmed := strings.TrimSpace(string(data))
		contentType := r.Header.Get("Content-Type")
		if trimmed != "" && (strings.Contains(contentType, "json") || r.Method == http.MethodPost) {
			if err := json.Unmarshal(data, &body); err != nil {
				return "", opts, err
			}
		}
	}

	text := firstNonEmpty(body.Text, body.Q, body.Input)
	q := r.URL.Query()
	if text == "" {
		text = firstNonEmpty(q.Get("text"), q.Get("q"), q.Get("input"))
	}

	if body.PreserveCase != nil {
		opts.PreserveCase = *body.PreserveCase
	}
	if body.PreserveCaseCamel != nil {
		opts.PreserveCase = *body.PreserveCaseCamel
	}
	if body.MaxAlternatives != nil {
		opts.MaxAlternatives = *body.MaxAlternatives
	}
	if body.MaxAlternativesAlt != nil {
		opts.MaxAlternatives = *body.MaxAlternativesAlt
	}

	if v := q.Get("preserve_case"); v != "" {
		opts.PreserveCase = parseBool(v, opts.PreserveCase)
	}
	if v := q.Get("preserveCase"); v != "" {
		opts.PreserveCase = parseBool(v, opts.PreserveCase)
	}
	if v := q.Get("max_alternatives"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			opts.MaxAlternatives = n
		}
	}
	if v := q.Get("maxAlternatives"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			opts.MaxAlternatives = n
		}
	}
	if v := q.Get("alternatives"); v != "" {
		if !parseBool(v, true) {
			opts.MaxAlternatives = 1
		}
	}
	if opts.MaxAlternatives < 1 {
		opts.MaxAlternatives = 1
	}
	if opts.MaxAlternatives > 20 {
		opts.MaxAlternatives = 20
	}
	return text, opts, nil
}

func dictionaryHandler(d *Dictionary) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
			return
		}
		q := normalizeKey(r.URL.Query().Get("q"))
		direction := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("direction")))
		if direction == "" {
			direction = "both"
		}
		limit := 50
		if v := r.URL.Query().Get("limit"); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		if limit < 1 {
			limit = 1
		}
		if limit > 500 {
			limit = 500
		}

		type item struct {
			Dialect  string `json:"dialect"`
			Standard string `json:"standard"`
		}
		items := make([]item, 0, limit)
		for _, e := range d.Entries {
			if q != "" {
				dialectMatch := strings.Contains(normalizeKey(e.Dialect), q)
				standardMatch := strings.Contains(normalizeKey(e.Standard), q)
				switch direction {
				case "standard":
					if !standardMatch {
						continue
					}
				case "dialect":
					if !dialectMatch {
						continue
					}
				default:
					if !dialectMatch && !standardMatch {
						continue
					}
				}
			}
			items = append(items, item{Dialect: e.Dialect, Standard: e.Standard})
			if len(items) >= limit {
				break
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"query":     q,
			"direction": direction,
			"count":     len(items),
			"entries":   items,
		})
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func parseBool(s string, def bool) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	}
	return def
}
