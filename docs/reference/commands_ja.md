# コマンド

> 英語版が正本（`commands.md`）。この文書はその日本語版。

ファイルの形式は `file-format_ja.md`。この文書の「組」「ルート」も、そちらの意味で使う。

## 共通

- **言語**：言語を扱うコマンドは、すべて引数で言語を指定できる
  - `new` は、言語をいくつも並べられる（`fnaze new en ja`）。省くと、いちばん新しい組と同じ言語。組が無ければ `FNAZE_LANG` の言語、それも無ければ `en`
  - ほかのコマンドは、言語を1つ指定できる（`--lang <言語>`）。省くと `FNAZE_LANG`、それも無ければ `en`
  - `FNAZE_LANG` には、`,` で区切っていくつかの言語を書ける（`FNAZE_LANG=ja,en`）。名前の前後の空白と、空の名前は無視する。`new` は書いた言語すべてを、ほかのコマンドは最初の言語を使う
  - コマンドは、指定した言語のファイルだけを扱う。ほかの言語のファイルとそろえることはしない。言語ごとのずれは `status` で分かる
  - `file-format_ja.md` の形（ASCII の英字と数字）に合わない言語の名前は、エラー
- **対象の組**：組を指定できるコマンドで省くと、いちばん新しい組（名前の順でいちばん後ろ）を使う。指定は、組の `<日時>_<UUID>`、またはその先頭の一部（1つに決まる長さ）
  - 組が1つも無いとき、`status` と `log` は何も書き出さない（`status` は、形に合わない名前のファイルは報告する）。`show`・`set`・`rm` は失敗する
- **`-C <ディレクトリ>`**：`completion` のほかのすべてのコマンドは、そのディレクトリで動かしたのと同じに動く（そこから上へ `.fnaze/` を探す。`init` はそこに作る）
- **`--json`**：`new`・`set`・`rm`・`status`・`show`・`log` は、結果を JSON で書き出す（下の「JSON」）。MCP の道具も同じ JSON を返す
- `-C`・`--json` は、下の各コマンドの使い方の行には書かない
- **使い方とバージョン**：`fnaze --help`（`fnaze <コマンド> --help`）で使い方を、`fnaze --version` でバージョンを書き出す
- **終了コード**：成功は `0`。`status --require` で報告があるときだけ `1`。エラー（コマンド行の誤りを含む）は `2`
- **テキストの出力**（`--json` が無いとき）：パスは今のディレクトリからの相対で、そのまま開ける（JSON ではルートからの相対）。色は付けない。`init`・`set`・`rm` は、成功したときは何も書き出さない
- **エラー**は、標準エラーに `fnaze: <メッセージ>` で書き出す
- fnaze は、`.fnaze/` の中のほかは書かない。ほかのプログラム（git など）を呼ばない

## JSON

- インデントは空白2つ。パスはルートからの相対で、どの OS でも `/` で区切る。ディレクトリは `/` で終わる
- キーはいつも出す。空の文字列は `""`、空の一覧は `[]`。`detail` は1つの文字列で、改行は `\n`
- 日時は UTC の RFC 3339（`2026-11-06T04:12:00Z`）
- エラーは JSON にしない。`--json` が無いときと同じく、標準エラーに書き出す

| コマンド | 出力 |
|---|---|
| `new` | `{"group", "files": [{"lang", "path"}]}` |
| `set` | `{"group", "lang", "path", "desc", "detail"}`：書いたあとのエントリ。`--json` が無ければ、`set` は何も書き出さない |
| `rm` | `{"group", "lang", "path", "removed"}`：`removed` は消したエントリの数（子と、上の骨組みの行を含む）。`--json` が無ければ、`rm` は何も書き出さない |
| `status` | `{"items": [{"group", "file", "line", "kind", "message"}]}`：`kind` は `format`・`empty-title`・`empty-desc`・`lang-mismatch`。1つの行の話でなければ `line` は `0` |
| `show` | `{"group", "lang", "file", "title", "entries": [{"name", "path", "desc", "detail", "children": [...]}]}` |
| `log` | `{"groups": [{"group", "datetime", "langs", "title", "entries"}]}`：新しい順。`entries` はエントリの数 |

## `fnaze init`

今のディレクトリ（`-C` を渡したときはそのディレクトリ）に `.fnaze/changes/` を作る。そこか、その上のどこかのディレクトリに `.fnaze/` があれば失敗する（`.fnaze/` はプロジェクトに1つだけ）。

## `fnaze new [言語…]`

新しい組を作る。言語ごとに1つずつ（並べた言語。省いたときは下のとおり）、同じ日時と UUID でファイルを作る。

```
fnaze new ja           # <日時>_<UUID>_ja.txt
fnaze new en ja        # <日時>_<UUID>.txt と <日時>_<UUID>_ja.txt
fnaze new              # いちばん新しい組と同じ言語
```

- **言語を省くと**、いちばん新しい組と同じ言語で作る。組が1つも無ければ、`FNAZE_LANG` の言語すべて（それも無ければ `en`）で作る
- **`en` も、並べたときだけ作る**
- 大文字と小文字だけが違う2つの言語（`ja` と `JA`）を並べると失敗する（`file-format_ja.md`）
- **中身**：言語ごとに、その言語のいちばん新しいファイルから、題・desc・detail を除いたものを作る
  - 残すもの：パスの行と、そのインデント
  - 除くもの：題の中身（`# ` だけ残す）、desc（タブごと）、detail の行
  - その言語のファイルがまだ無ければ、`# ` だけのファイルを作る
- 作ったファイルのパスを、今のディレクトリからの相対で、1行に1つ書き出す

```
$ fnaze new en ja
.fnaze/changes/20261108T010203Z_00010203-0405-4607-8809-0a0b0c0d0e0f.txt
.fnaze/changes/20261108T010203Z_00010203-0405-4607-8809-0a0b0c0d0e0f_ja.txt
```

例：`ja` のいちばん新しいファイルが

```
# 認証の期限を延長に変えた
src/
	auth/	期限を延長に変えた
		session.go	アクセスのたびに期限を延ばす
			| 短すぎてセッションが切れる報告があった
```

なら、`fnaze new ja` が作るファイルは

```
# 
src/
	auth/
		session.go
```

## `fnaze set [--lang <言語>] [--group <組>] <path> <desc> [--detail <detail>]…`

対象の組の、その言語（「共通」の「言語」）のファイルに、エントリを1つ書く。

- `--group <組>` で組を指定できる（省くと、いちばん新しい組）
- パスは、ルートからのパスとして書いたまま読む。末尾に `/` があればディレクトリ。`file-format_ja.md` で使えるパスに限る。desc に改行は書けない
- そのパスのエントリがあれば desc を置き換え、`--detail` を渡したときは detail も置き換える
- 無ければ、ファイルにある、いちばん深い上のディレクトリのエントリの下に、兄弟の末尾に足す。途中のディレクトリのエントリが無ければ、1段ずつ骨組みの行として足す
- `--detail` 1つが detail の1行。値の中の改行でも行を分ける。渡した値がすべて空なら、detail を消す
- その言語のファイルが組に無いとき、ファイルに形の誤りがあるときは失敗する
- ファイルは、決まった形（`file-format_ja.md` の「中身」）で書き直す
- 同じ組のほかの言語のファイルは変えない（パスの行のずれは `status` で分かる）

## `fnaze rm [--lang <言語>] [--group <組>] <path>`

対象の組の、その言語のファイルから、エントリを1つ消す。子と detail も一緒に消す。`--group` は `set` と同じ。

- 消したことで子が無くなった上のディレクトリのエントリのうち、desc も detail も無いもの（骨組みの行）も消す
- その言語のファイルが組に無いとき、ファイルに形の誤りがあるとき、そのパスのエントリが無いときは失敗する
- ファイルは、決まった形で書き直す

## `fnaze status [--require] [--all | 組]`

組を検査する。組の、すべての言語のファイルを見る。

報告すること：

| 種類 | 内容 |
|---|---|
| 形の誤り | 題の行が無い・インデントの崩れ・同じパスが2つ・使えないパス・子を持つファイルのエントリ・BOM・置き場所の形に合わない名前のファイル |
| 空の題 | `#` だけ、または `# ` だけの題 |
| 空の desc | 子の無いエントリで、desc の無いもの |
| 言語ごとのずれ | 同じ組の言語ごとのファイルで、パスの行（パスとその順）が違うもの |

- 何も無ければ何も書き出さない。あれば、ファイルの順、行の順に、1つにつき1行、`<ファイル>:<行>: <種類>: <メッセージ>` の形で書き出す（行が 0 のときは `:<行>` を省く）。ファイルは今のディレクトリからの相対、種類は JSON と同じ

```
.fnaze/changes/20261108T010203Z_00010203-0405-4607-8809-0a0b0c0d0e0f.txt:3: empty-desc: src/a.go has no desc
.fnaze/changes/notes.txt: format: the name does not have the form <datetime>_<UUID>[_<language>].txt
```

- 終了コードは、`--require` を付けないときは、報告があっても 0。付けたときは、報告が1つでもあれば 1。エラーは、どのコマンドとも同じく 2
- `--all` を付けると、すべての組を検査する。組と一緒には指定できない
- `.fnaze/changes/` の中の、置き場所の形に合わない名前のファイルは、どの組を検査しても報告する（JSON では `group` が `""`）

## `fnaze show [--lang <言語>] [組]`

組の、その言語のファイルを、ツリーで見せる。題を先頭に出し、パスと desc の列をそろえ、detail は desc の下に出す。その言語のファイルが組に無ければ失敗する。

- 題のあとに空行を1つ入れる。インデントは1段につき空白2つ。パスはファイルに書いたまま
- desc は、いちばん幅の広いパスのあとに空白2つを置いた列から始める。幅は表示幅で数える（全角は2）
- detail の行は、desc の列に、`| ` を付けて出す

```
# Extend the session lifetime on access instead of fixing it

src/
  auth/           Change the session lifetime from fixed to extended
    session.go    Extend the lifetime on every access
    *_test.go     Add tests for the extension
config/auth.yaml  Set the default session.ttl to 30m
                  | Users reported sessions expiring too soon
docs/old.md       Removed; its content moved to docs/auth.md
```

## `fnaze log [--lang <言語>]`

組を、新しい順に1行ずつ並べる。組の短い名前、日時、組にある言語、エントリの数、その言語の題を出す。その言語のファイルが無い組は、題を空にする。日時は利用者のタイムゾーンで、分まで出す（JSON では UTC）。

- 組の短い名前は、組の名前の先頭で、ほかの組と区別できる長さのもの。少なくとも、日時・`_`・UUID の4文字。組を指定できるところなら、どこにでも渡せる
- 言語は `,` でつなぐ。列はそろえる（エントリの数は右にそろえる）

```
20261107T021530Z_2c4c  2026-11-07 11:15  en,ja  3  Log session extensions
20261107T003426Z_8980  2026-11-07 09:34  en,ja  6  Extend the session lifetime on access instead of fixing it
```

（例は `TZ=Asia/Tokyo` のもの）

## `fnaze mcp [--allow-write]`

MCP サーバーとして、標準入出力で動く。AIエージェントが道具として呼べる。

- サーバーは言語を持たない。道具ごとに、コマンドと同じ形で言語を引数に取る（`new` はいくつも、ほかは1つ）。省いたときの決め方も、コマンドと同じ
- 道具は、コマンドに `--json` を付けたときと同じ JSON を返す。エラーは道具のエラーとして、コマンドと同じ文言で返す
- 読む道具は、コマンドと同じ名前：`show`・`status`・`log`
- 書く道具は、別の名前にする。AIが、ファイルを作る・消す道具と取り違えないようにするため：`new_group`（`new`）・`set_description`（`set`）・`remove_description`（`rm`）。`--allow-write` を付けたときだけ出す
- ルートは、呼ぶたびに、`-C <dir>` かサーバーを動かしたディレクトリから探す。サーバーが動いている間に作った・変えた `.fnaze/` も見える

引数の意味は、コマンドのオプションと同じ。`?` を付けたものは省ける。省いたときも、コマンドと同じ。

| 道具 | 引数 | コマンド |
|---|---|---|
| `show` | `lang?`、`group?` | `--lang`、`[組]` |
| `status` | `group?`、`all?`（真偽） | `[組]`、`--all` |
| `log` | `lang?` | `--lang` |
| `new_group` | `langs?`（文字列の配列） | `[言語…]` |
| `set_description` | `lang?`、`group?`、`path`、`desc`、`detail?`（文字列の配列） | `--lang`、`--group`、`<パス> <desc>`、`--detail`… |
| `remove_description` | `lang?`、`group?`、`path` | `--lang`、`--group`、`<パス>` |

- `detail`：文字列1つが1行で、文字列の中の改行は次の行になる。省くと detail を保つ。`[]` か、空の文字列だけなら消す
- `status` に `require` は無い。道具には終了コードが無く、`items` が空なら報告は無い
- 結果は、`--json` の JSON とバイトまで同じテキストで、structured content にも同じ JSON を入れる。出力のスキーマは無い
- エラーは道具のエラーで、テキストはコマンドのメッセージから `fnaze: ` を除いたもの

## `fnaze completion <シェル>`

シェルの補完のスクリプトを書き出す。シェルは `bash`・`zsh`・`fish`・`powershell`。

```
fnaze completion bash >~/.local/share/bash-completion/completions/fnaze
fnaze completion zsh >"${fpath[1]}/_fnaze"
fnaze completion fish >~/.config/fish/completions/fnaze.fish
fnaze completion powershell >>$PROFILE
```

補完するもの：

- コマンドの名前（説明として、何をするかを添える）と、それぞれのフラグ
- 組（`set`・`rm` の `--group`、`status`・`show` の引数）：今ある組の `<日時>_<UUID>`。新しい順に出し、説明として、その言語の題を添える（その言語のファイルが無い組には添えない）
- 言語（`new` の引数と `--lang`）：今ある組に使われている言語の名前。いちばん新しい組の並び、続けて古い組にだけあるもの、の順。`new` では、すでに書いた言語を除く
- パス（`set`・`rm` の `<path>`）：対象の組の、その言語のファイルにあるエントリのパス。1段ずつ出す。ディレクトリは `/` で終わり、空白を入れずに続けられる。エントリには、説明として desc を添える

```
$ fnaze rm src/auth/<TAB>
src/auth/            Auth
src/auth/session.go  Extend
src/auth/*_test.go   Tests
```

- 説明が表示幅で60桁より長いときは切り、`…` で終える
- 候補が無いところ（`-C` の値、`set` の新しいパス）では、シェルがファイルを補う

ここでの言語は、補完しているコマンド行にすでにある `--lang`。無ければ `FNAZE_LANG` の最初の言語、それも無ければ `en`。

組・言語・パスは、補完のたびに fnaze を呼んで `.fnaze/` から求める（ほかのプログラムは呼ばない）。コマンド行に `-C <ディレクトリ>` があれば、そこから `.fnaze/` を探す。
