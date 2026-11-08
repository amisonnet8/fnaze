# 命名規則

## 製品名

- **fnaze**。文中でもコマンドでも、すべて小文字で書く
- データの置き場は`.fnaze/`（説明のファイルは`.fnaze/changes/`）、環境変数は`FNAZE_LANG`

## 用語

出所は`docs/reference/file-format.md`・`commands.md`と`docs/design/overview.md`。**用語を変えたり足したりしたら、docs とこの表を同じ変更の中で直す**（mtqg の glossary にも登録してある）。英語は docs（英語が正本）での言い方、CLI・JSON は出力やオプションでの表記。

| 用語 | 英語 | CLI・JSON での表記 | 意味 |
|---|---|---|---|
| 説明のファイル | description file | `file` | `.fnaze/changes/<日時>_<UUID>[_<言語>].txt`。変更のまとまり1つを、1つの言語で説明する |
| まとまり（変更のまとまり） | set of changes | — | 説明のファイル1つが説明する変更。何を1つとするか（コミット1つなど）は使う人が決める |
| 組 | group | `group`、`--group` | 同じ`<日時>_<UUID>`を持つ、言語ごとの説明のファイルの集まり。指定は`<日時>_<UUID>`か、その先頭の一部 |
| ルート | root | — | `.fnaze/`のあるディレクトリ。パスはここからの相対 |
| 題 | title | `title` | 説明のファイルの1行目（`# `で始まる） |
| エントリ | entry | `entries` | 2行目からの1行。パスと、あれば desc |
| desc | desc | `desc` | エントリの、何を・なぜ変えたかの1行 |
| detail | detail | `detail` | `\| `で始まる行。直前のエントリの補足 |
| 骨組みの行 | skeleton line | — | desc の無い、子を持つディレクトリのエントリ。深い場所に書くための行 |
| パターン | pattern | — | `*`を含むパスの最後の部分。書いたディレクトリの直下だけに当たる |
| 言語 | language | `lang`・`langs`、`--lang`、`FNAZE_LANG` | 説明のファイルを分ける、ただの文字列。`en`だけはファイル名に接尾辞を付けない |
| 形の誤り | format error | `format` | `status`の報告の種類 |
| 空の題 | empty title | `empty-title` | 同上。`# `だけの題 |
| 空の desc | empty desc | `empty-desc` | 同上。子の無いエントリで desc の無いもの |
| 言語ごとのずれ | language mismatch | `lang-mismatch` | 同上。同じ組の言語ごとのファイルで、パスの行が違うもの |

## コマンド・道具の名前

- CLI：`init`・`new`・`set`・`rm`・`status`・`show`・`log`・`mcp`・`completion`（`docs/reference/commands.md`）
- MCP：読む道具は CLI と同じ名前（`show`・`status`・`log`）。書く道具だけ`new_group`・`set_description`・`remove_description`（ファイルを作る・消す道具と取り違えられないように）
- 共通のオプション：`-C <dir>`・`--json`
