# github.com/daqing/airway-moderation-plugin

An [Airway](https://github.com/daqing/airway) plugin that screens text against
a word list, as a service other services call over HTTP. One place owns the
word list — a JSON file operations edit and hot-reload — and every caller asks
it the same question, so a host and its clients never drift apart on what
counts as content that needs review.

Matching is Aho-Corasick over the whole list in a single pass, and the answer
names the words that hit and the category each belongs to, so a reviewer sees
*why* a message was held rather than just that it was.

## API

Mounted at `/api/v1/moderation`. The response is the framework's envelope —
`{"code": 0, "data": ..., "message": ""}` — and errors keep HTTP 200 with a
non-zero `code`, so a caller has to read the body:

| Code | Meaning |
| --- | --- |
| `30001` | The request cannot be screened (malformed body, content over 32 KiB). |
| `30002` | Missing or wrong shared secret. |
| `30003` | The word list could not be loaded. |

| Method | Path | Description |
| --- | --- | --- |
| `POST` | `/screen` | Body `{"content": "..."}`; answers `{"flagged": bool, "categories": [...], "words": [...]}`. Empty content never hits; a hit reports each word once, in the order the hits end. |
| `GET` | `/words` | The list in use, grouped by category, plus where it came from. |
| `POST` | `/reload` | Re-reads the word list file. Answers with the same body as `/words`. |
| `GET` | `/health` | Liveness probe, no secret required: `{status, words, source, version}`, where `source` is `file` or `embedded`. Reports no path and no words. |

```bash
curl -sX POST localhost:1905/api/v1/moderation/screen \
  -H 'Content-Type: application/json' \
  -H "X-Moderation-Secret: $MODERATION_SECRET" \
  -d '{"content":"有人在群里发广告，还提到网络赌博，大家小心"}'
# {"code":0,"data":{"flagged":true,"categories":["illegal"],"words":["网络赌博"]},"message":""}
```

### Authentication

`/screen`, `/words` and `/reload` are guarded by the shared secret in
`MODERATION_SECRET`, which callers present as `X-Moderation-Secret`. Leave the
variable unset on a local checkout and the endpoints stay open; the plugin
logs a warning at boot when it does. `/health` is never guarded, so a
container probe needs no secret.

## The word list

```json
{
  "categories": {
    "illegal": ["网络赌博"],
    "porn": ["裸聊"],
    "weapons": ["仿真枪"]
  }
}
```

`MODERATION_WORDS_FILE` names the file, defaulting to
`config/moderation_words.json`. Paths are relative to the working directory
the host runs in. Words are lower-cased and trimmed on load, duplicates and
empty categories are dropped, and a word listed under two categories stays
with the first one alphabetically, so a hit always names exactly one category.

A path that is not there falls back to the list built into the plugin, which
keeps a checkout that never ran `plugin:install` working; `GET /health` and
`GET /words` both report which list is in use. A file that *is* there but
cannot be parsed fails the host boot, and a `/reload` of one fails with
`30003` while the list already in use keeps screening — an operator's typo
never disarms the service.

Editing the file and calling `/reload` applies it without a restart:

```bash
$EDITOR config/moderation_words.json
curl -sX POST localhost:1905/api/v1/moderation/reload -H "X-Moderation-Secret: $MODERATION_SECRET"
```

### Where the words come from

The built-in list is generated from the high-signal categories of
[Sensitive-lexicon](https://github.com/konsheng/Sensitive-lexicon), an MIT
keyword collection, and holds 15,652 words in eight categories: `corruption`
176, `illegal` 78, `political` 8, `porn` 406, `spam` 46, `url` 14,370,
`violent` 159, `weapons` 409. `THIRD_PARTY_NOTICES.md` records the upstream
revision, the license, the upstream files this list comes from, and the ones
left out with the reason for each. Nothing follows upstream on its own: moving
to a newer revision means checking it out, rerunning the generator, and reading
the diff.

That collection was written for search engines, where 兼职, 自拍, 起诉 and a
bare `www` are useful keywords. Here every hit holds a message for a human, so
the generator drops what ordinary chat uses — first mechanically (entries
shorter than two characters, and pure-ASCII entries too short to mean anything
on their own), then against a curated exclusion list,
`tools/lexicon/exclude.txt`, which lives in this repository and exists to be
edited. A word an operator disagrees with comes out of that file, or goes into
`tools/lexicon/starter_words.json` to join the plugin's own hand-picked words,
and the list follows:

```bash
go run ./tools/lexicon -src /path/to/Sensitive-lexicon/Vocabulary \
  -out install/lib/moderation/default_words.json
go run ./tools/lexicon -src /path/to/Sensitive-lexicon/Vocabulary \
  -out install/host/config/moderation_words.json
```

The same content then has to be copied to the host project's
`config/moderation_words.json`, which is the file a running host actually
reads. `-check` compares a file against what the sources generate instead of
writing it, so all three can be verified in step. The generator reports what it
kept per category, what it dropped and why, and any exclusion entry that
matched nothing — meaning upstream no longer carries that word.

Coverage is not correctness: the list tells a reviewer what to look at, not
what is true, and words it does not hold still need an operator's eye. Every
hit names the word that matched, so the review queue itself says what to
tighten.

## Use in a host application

```bash
go get github.com/daqing/airway-moderation-plugin
```

Then enable it with a blank import in the host's `plugins.go`:

```go
import (
	_ "github.com/daqing/airway-moderation-plugin"
)
```

and copy the starter word list into the host project:

```bash
go run . plugin:install github.com/daqing/airway-moderation-plugin
```

`plugin:install` writes `install/host/config/moderation_words.json` to
`config/moderation_words.json` in the host project and leaves an existing file
alone, so it is safe to run again after an upgrade. The plugin registers no
migrations — it stores nothing.

Callers that would rather not speak HTTP can use the library the endpoints are
built on, `install/lib/moderation`, which is compiled into the host binary:

```go
service, err := moderation.New(moderation.WordsFile())
result := service.Screen(content)
```

## Callers

The [Airway IM plugin](https://github.com/daqing/airway-im-plugin) screens every
message it accepts by calling `POST /screen` at `IM_MODERATION_URL`, presenting
the same `MODERATION_SECRET` as `X-Moderation-Secret`. It is fail-closed: a
message is held for operator review when this service cannot answer, so while it
is unreachable every send waits for a human instead of reaching the
conversation. Its review queue shows the words `/screen` reported.

## Configuration

| Variable | Default | Description |
| --- | --- | --- |
| `MODERATION_SECRET` | unset | Shared secret for `/screen`, `/words` and `/reload`. Unset leaves them open. |
| `MODERATION_WORDS_FILE` | `config/moderation_words.json` | Word list to screen against. A missing file falls back to the built-in list. |

## Develop

```bash
go get github.com/daqing/airway@latest
go mod tidy
go test ./...
```

`install/lib/moderation` holds the word list, the matcher and the screening
service and depends on nothing but the standard library;
`install/lib/api/moderation_api` serves them. Its tests include two that run the
shipped list over a corpus of ordinary messages and one of messages that must
be caught, so a word that starts holding harmless chat back shows up as a
failing test. `tools/lexicon` generates the list itself; see
`THIRD_PARTY_NOTICES.md`.

## Install layout

Code under `install/lib/` is compiled into the host binary through the blank
import and is never copied. Everything under `install/host/` is copied into
the host project by `plugin:install` — here, the starter word list — and
`install/ignore/` never leaves the plugin checkout. `install/deps/` carries
companion services for hosts that need them; this plugin has none.

See https://github.com/daqing/airway/blob/main/docs/plugin.md for the full
plugin guide.
