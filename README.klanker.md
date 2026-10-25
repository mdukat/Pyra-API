# Pyra — standard Polish → Poznań dialect HTTP translator

A small, dependency-free Go HTTP server that translates standard Polish text
into the Poznań dialect (`gwara poznańska`). The dictionary is embedded from
[`slownik.md`](slownik.md) (3,266 entries) and indexed at start-up with a
rule-based Polish morphology engine, so inflected forms are translated too:

```
ziemniak  -> pyra
ziemniaka -> pyry
ziemniakiem -> pyrą
ziemniaków -> pyr
śpi       -> garuje
```

See [`API.md`](API.md) for the full HTTP API schema.

## Build and run

```bash
go build -o pyra-server .
./pyra-server                 # listens on :8080
./pyra-server -addr :9000     # custom address
```

## Quick test

```bash
curl -s 'http://localhost:8080/translate?text=Ziemniaki%20s%C4%85%20w%20kuchni.'
# {"input":"Ziemniaki są w kuchni.","translation":"Pyry są wew kuchni.",...}

curl -s -X POST http://localhost:8080/translate \
  -H 'Content-Type: application/json' \
  -d '{"text":"Nie ma chleba."}'
# {"input":"Nie ma chleba.","translation":"Nie ma byba.",...}
```

## Endpoints

| Method | Path | Description |
| --- | --- | --- |
| `GET`/`POST` | `/translate?text=...` | Translate text (query string or JSON body). |
| `GET` | `/dictionary?q=...` | Search raw dictionary entries. |
| `GET` | `/healthz` | Liveness probe. |
| `GET` | `/` | Service metadata. |

## Development

```bash
go test ./...
```

The tests parse the embedded dictionary and cover inflection, sentence
translation, capitalisation, alternatives and the HTTP handlers.
