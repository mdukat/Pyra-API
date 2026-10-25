# Pyra Translation API — API Schema

HTTP API that translates **standard Polish** text into the **Poznań dialect**
(`gwara poznańska`). The server is a single Go binary; it embeds
[`slownik.md`](slownik.md) (3,266 dictionary entries) and builds an inflected
reverse index at start-up, so the API understands word variants such as
`ziemniak`, `ziemniaka`, `ziemniaki`, `ziemniakiem` and returns
`pyra`, `pyry`, `pyrą`, … as appropriate.

* Base URL (default): `http://localhost:8080`
* All responses are `application/json; charset=utf-8`.
* All request/response text is UTF-8.
* No authentication is required.

---

## 1. Endpoints at a glance

| Method | Path | Description |
| --- | --- | --- |
| `GET` | `/translate` | Translate text supplied in the query string. |
| `GET` | `/translate` (JSON body) | Translate text supplied in a JSON body (non-standard but supported). |
| `POST` | `/translate` | Translate text supplied in a JSON body. |
| `GET` | `/dictionary` | Search the raw dictionary entries. |
| `GET` | `/healthz` | Liveness probe. |
| `GET` | `/` | Service metadata and endpoint list. |
| `OPTIONS` | any | CORS pre-flight. |

---

## 2. `GET /translate`

### Query parameters

| Name | Type | Required | Default | Description |
| --- | --- | --- | --- | --- |
| `text` | string | yes* | — | Text to translate. `q` and `input` are accepted aliases. |
| `q` | string | alias | — | Alias for `text`. |
| `input` | string | alias | — | Alias for `text`. |
| `preserve_case` / `preserveCase` | bool | no | `true` | Copy the capitalisation of the source word (`Ziemniaki` → `Pyry`, `ZIEMNIAKI` → `PYRY`). |
| `max_alternatives` / `maxAlternatives` | int | no | `5` | Maximum number of alternative translations per matched segment (1–20). |
| `alternatives` | bool | no | `true` | `false` is a shorthand for `max_alternatives=1`. |

\* `text` must be provided either in the query string or in a JSON body.

### Example

```
GET /translate?text=Ziemniaki%20s%C4%85%20w%20kuchni.&max_alternatives=3
```

---

## 3. `POST /translate` (and `GET` with a JSON body)

### Request body

```json
{
  "text": "Mam ziemniaki.",
  "preserve_case": true,
  "max_alternatives": 3
}
```

| Field | Type | Required | Default | Description |
| --- | --- | --- | --- | --- |
| `text` | string | yes* | — | Text to translate. `q` and `input` are accepted aliases. |
| `q` | string | alias | — | Alias for `text`. |
| `input` | string | alias | — | Alias for `text`. |
| `preserve_case` / `preserveCase` | bool | no | `true` | See above. |
| `max_alternatives` / `maxAlternatives` | int | no | `5` | See above. |

\* Query parameters are still read, but a JSON body takes precedence for the
fields it sets.

### Example

```bash
curl -s -X POST http://localhost:8080/translate \
  -H 'Content-Type: application/json' \
  -d '{"text":"Nie ma chleba.","preserve_case":true}'
```

---

## 4. Translation response

Both `GET` and `POST /translate` return the same shape.

```json
{
  "input": "Ziemniaki są w kuchni.",
  "translation": "Pyry są wew kuchni.",
  "segments": [
    {
      "source": "Ziemniaki",
      "translation": "Pyry",
      "matched": true,
      "standard": "ziemniak",
      "dialect": "pyra",
      "alternatives": ["Pyrki"]
    },
    {
      "source": "są",
      "translation": "są",
      "matched": false
    },
    {
      "source": "w",
      "translation": "wew",
      "matched": true,
      "standard": "w",
      "dialect": "wew"
    },
    {
      "source": "kuchni",
      "translation": "kuchni",
      "matched": false
    }
  ],
  "stats": {
    "words": 4,
    "translated_words": 2,
    "untranslated_words": 2,
    "matches": 2
  }
}
```

### Response fields

| Field | Type | Description |
| --- | --- | --- |
| `input` | string | The original input text. |
| `translation` | string | The translated text, punctuation and spacing preserved. |
| `segments` | array | Ordered list of translated/untranslated pieces covering the whole input. |
| `segments[].source` | string | Original substring (word or phrase). |
| `segments[].translation` | string | Translated substring. For unmatched segments this equals `source`. |
| `segments[].matched` | bool | `true` when the dictionary produced a translation. |
| `segments[].standard` | string | Matched standard-Polish lemma (omitted when `matched=false`). |
| `segments[].dialect` | string | Matched dialect lemma (omitted when `matched=false`). |
| `segments[].alternatives` | string[] | Other valid dialect forms, capitalisation already applied (omitted when empty). |
| `stats.words` | int | Number of words found in the input. |
| `stats.translated_words` | int | Number of words covered by matches. |
| `stats.untranslated_words` | int | Number of words left unchanged. |
| `stats.matches` | int | Number of matched segments (phrases count once). |

---

## 5. `GET /dictionary`

Search the raw `slownik.md` entries.

### Query parameters

| Name | Type | Required | Default | Description |
| --- | --- | --- | --- | --- |
| `q` | string | no | `""` | Case-insensitive substring; when empty the first entries are returned. |
| `direction` | enum | no | `both` | One of `both`, `standard`, `dialect`. |
| `limit` | int | no | `50` | 1–500. |

### Example response

```json
{
  "query": "pyra",
  "direction": "dialect",
  "count": 2,
  "entries": [
    { "dialect": "pyra", "standard": "ziemniak" },
    { "dialect": "pyrka", "standard": "ziemniak" }
  ]
}
```

---

## 6. `GET /healthz`

```json
{ "status": "ok", "dictionary_entries": 3266 }
```

---

## 7. `GET /`

```json
{
  "name": "Pyra Translation API",
  "description": "Translates standard Polish text into the Poznań dialect.",
  "version": "1.0.0",
  "dictionary_entries": 3266,
  "endpoints": {
    "translate": "GET/POST /translate?text=...",
    "dictionary": "GET /dictionary?q=...",
    "health": "GET /healthz"
  },
  "example": "/translate?text=Ziemniaki%20s%C4%85%20w%20kuchni."
}
```

---

## 8. Errors

Errors use the same JSON content type and a non-2xx status.

```json
{
  "error": {
    "code": "missing_text",
    "message": "provide the text to translate as ?text=..., ?q=..., ?input=... or a JSON body {\"text\":\"...\"}"
  }
}
```

| Status | `code` | Meaning |
| --- | --- | --- |
| `400` | `missing_text` | No `text`/`q`/`input` was provided. |
| `400` | `invalid_request` | The JSON body could not be decoded. |
| `404` | `not_found` | Unknown path. |
| `405` | `method_not_allowed` | Wrong HTTP method for the endpoint. |

---

## 9. Translation behaviour (contract)

1. **Greedy longest match.** The translator scans the sentence left to right
   and always prefers the longest dictionary phrase that fits, so multi-word
   entries win over single words. Phrases are not matched across punctuation.
2. **Inflection.** For every matched word the engine determines the
   grammatical form of the standard-Polish input (case, number, gender,
   person, tense) and generates the corresponding form of the dialect word.
   Examples:
   * `ziemniak` → `pyra`
   * `ziemniaka` → `pyry`
   * `ziemniakiem` → `pyrą`
   * `ziemniaków` → `pyr`
   * `pracuje` → `funktuje`
   * `śpi` → `garuje`
3. **Alternatives.** Many standard words have several dialect equivalents
   (e.g. `rower` → `koło`, `kółko`, `kufta`, `kuwta`). The best candidate is
   returned in `translation`; the rest are listed in
   `segments[].alternatives`.
4. **Unknown words and punctuation** are copied unchanged. Whitespace and
   punctuation are preserved byte-for-byte.
5. **Capitalisation** follows the first word of each matched segment:
   lower-case stays lower-case, `Ziemniaki` → `Pyry`, `ZIEMNIAKI` → `PYRY`.
6. **Direction is one-way** (standard Polish → Poznań dialect). Input that is
   already dialect is generally left unchanged unless it happens to collide
   with a standard form.

---

## 10. Client implementation notes

* URL-encode `text` when using the query string (spaces as `%20` or `+`,
  Polish letters as UTF-8 percent-escapes). The server accepts either.
* Send `Accept: application/json`; the server always answers with JSON.
* CORS is enabled for browser clients (`Access-Control-Allow-Origin: *`).
* There is no request-size limit beyond 1 MiB for JSON bodies; keep
  `text` reasonably short (sentences/paragraphs, not whole books).
* Treat `segments` as optional: the simplest client only needs
  `translation`. Use `segments` to highlight which parts were translated.
* Empty or whitespace-only `text` is rejected with `400 missing_text`.

### Minimal client pseudocode

```text
POST /translate
Content-Type: application/json

{"text": userText, "preserve_case": true, "max_alternatives": 3}

-> 200 {"translation": "...", "segments": [...], "stats": {...}}
```

```js
const res = await fetch("http://localhost:8080/translate", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ text, preserve_case: true }),
});
if (!res.ok) throw new Error((await res.json()).error.message);
const { translation, segments } = await res.json();
```
