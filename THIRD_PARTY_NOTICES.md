# Third-party notices

## Sensitive-lexicon

The moderation word list at `install/lib/moderation/default_words.json` (and its
copy at `install/host/config/moderation_words.json`) is generated from
[Sensitive-lexicon](https://github.com/konsheng/Sensitive-lexicon), a keyword
collection maintained by Konsheng. It is used under the MIT license, reproduced
in full at the end of this file.

Upstream revision this list was generated from:

```
d967c30b053fa40b06c5a0dddf0be493f2dfae46   (2026-08-17)
```

Nothing else from that repository is vendored here: the upstream `.txt` files
are not part of this repository, and this list is not updated automatically.
Moving to a newer upstream revision means checking it out, rerunning the
generator below, and reviewing the diff of the generated JSON.

### What is used

The generator reads eight of the upstream files, one category each, and keeps
the entries that pass the filters below. Counts are the entries that survived
all filtering:

| Upstream file | Category | Words kept |
| --- | --- | ---: |
| `色情词库.txt` | `porn` | 400 |
| `色情类型.txt` | `porn` | 270 |
| `暴恐词库.txt` | `violent` | 162 |
| `涉枪涉爆.txt` | `weapons` | 406 |
| `广告类型.txt` | `spam` | 55 |
| `非法网址.txt` | `url` | 14,370 |
| `贪腐词库.txt` | `corruption` | 176 |
| `其他词库.txt` | `illegal` | 69 |

Those per-file counts overlap: the two `porn` files carry many of the same
words, and a word that two categories both claim stays with the first one
alphabetically. After that de-duplication and the merge with this plugin's own
starter words, the list holds 15,652 words:

| Category | Words |
| --- | ---: |
| `corruption` | 176 |
| `illegal` | 78 |
| `political` | 8 |
| `porn` | 406 |
| `spam` | 46 |
| `url` | 14,370 |
| `violent` | 159 |
| `weapons` | 409 |

The eight `political` words and the other 20 words of the plugin's original
starter list do not come from upstream; they are this plugin's own, and they
are kept in `tools/lexicon/starter_words.json`.

### What is not used, and why

Upstream is a keyword collection for search engines and content sites. A chat
filter sends every hit to a human review queue, so a word earns a place here
only when its ordinary use is rare. These files were left out entirely:

| Upstream file | Why |
| --- | --- |
| `反动词库.txt`, `政治类型.txt` | broad political vocabulary |
| `民生词库.txt` | everyday livelihood vocabulary |
| `GFW补充词库.txt` | network censorship keywords, not chat content |
| `网易前端过滤敏感词库.txt`, `零时-Tencent.txt` | tokenizer keywords, mostly ordinary words |
| `COVID-19词库.txt` | pandemic vocabulary |
| `新思想启蒙.txt` | meme phrases |
| `补充词库.txt` | segmented sentence fragments |

From the eight files that are used, these entries were dropped:

| Filter | Entries dropped |
| --- | ---: |
| pure ASCII shorter than the category floor (6 characters, 8 for `url`) | 299 |
| carries a space or a pipe | 48 |
| listed in `tools/lexicon/exclude.txt` | 384 |
| shorter than 2 characters | 3 |

The exclusion list is this project's own judgement, not upstream's, and it is
meant to be edited: see "The word list" in `README.md`.

### Regenerating

```sh
go run ./tools/lexicon -src /path/to/Sensitive-lexicon/Vocabulary \
  -out install/lib/moderation/default_words.json
go run ./tools/lexicon -src /path/to/Sensitive-lexicon/Vocabulary \
  -out install/host/config/moderation_words.json
```

The same content also has to be written to the host project's word list file
(`config/moderation_words.json`), which is the file a running host actually
reads; the embedded copy only applies when that file is missing. `-check`
compares a file against what the sources generate instead of writing it, which
is how all three are verified to be in step.

---

## MIT License

Copyright (c) 2024~2099 Konsheng

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
