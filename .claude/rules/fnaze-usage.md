# fnazeの使い方

このリポジトリは、**コミットごとに、そのコミットの説明を fnaze 自身で書き、一緒にコミットする**（`.fnaze/changes/`）。人間は、説明の一覧と diff を並べてレビューする。言語は**英語（`en`）と日本語（`ja`）の両方**（q `564f68ac3b`）。英語のファイルは接尾辞なし（`<日時>_<UUID>.txt`）、日本語は`_ja.txt`。仕様は`docs/reference/commands_ja.md`と`file-format_ja.md`。

## 言語の設定

- crube（`.claude/rules/crube-usage.md`）と同じ形にする。**`FNAZE_LANG`は設定しない**（`.devcontainer/`にも`.claude/settings.json`にも置かない）。言語を省いたコマンドは英語（`en`）を使い、日本語は`--lang ja`で指定する
- 書くときは、**いつも`--lang en`・`--lang ja`を明示する**（手元の環境に`FNAZE_LANG`があっても、同じ結果になる）

## コミットのたびに

1. 入れるファイルを`git add`し、`git diff --cached --name-only`で変わったファイルを確かめる
2. `fnaze new en ja`で組を作る。言語ごとに、1つ前のその言語のファイルのパスの行が、骨組みとして写ってくる
3. `fnaze title --lang en <題>`と`fnaze title --lang ja <題>`で題を書く。題は、このまとまりで何をしたかの1行。英語と日本語で同じことを書く
4. 変わったファイル・ディレクトリごとに、`fnaze set --lang en <path> <desc> [--detail …]`と`fnaze set --lang ja <path> <desc> [--detail …]`で書く
   - desc は「何を・なぜ変えたか」の1行。経緯や補足は`--detail`
   - ツールを通して変わったファイル（mtqg の`.mtqg/journal.jsonl`、crube の`.crube/structure*.txt`、`go get`の`go.sum`など）も、そのパスをエントリにする。desc には、どのツールのどの操作で何が変わったかを、分かる範囲で書く（mtqg ならレコードの ID）。特定できなければ、親のディレクトリの desc にまとめる
   - 説明のファイル自身（`.fnaze/changes/…`）は、エントリに書かない
   - **2つの言語で、同じパスを同じ順に`set`する**。並びが違うと`status`が言語ごとのずれ（`lang-mismatch`）を報告する
5. このコミットで変わっていない、写ってきたパスは`fnaze rm --lang en <path>`と`fnaze rm --lang ja <path>`で消す
6. `fnaze status --require`を通し、説明のファイルを`git add .fnaze`してからコミットする
7. 置いた説明のファイルのパス（2つ）を、人間に伝える

- **説明のファイルを`Edit`・`Write`で直さない。** 題もエントリも、fnaze のコマンドで書く（fnaze を使って確かめるため）
- **pre-commit が止める**：`.githooks/pre-commit`（`docs/examples/pre-commit`と同じ中身。`core.hooksPath`は`.devcontainer/postCreate.sh`が設定する）が、説明のファイルを足さないコミットと、いちばん新しい組を`fnaze status --require`が報告するコミットを止める。止まったら、説明を書いて直す。`--no-verify`で飛ばさない

## 落とし穴

- **`-`で始まる desc・題は、オプションとして読まれる**（`unknown option --allow-write…`）。オプションは引数の前にも後ろにも置けるため。`fnaze set --lang ja -- <path> <desc>`のように、`--`の後ろに置く
- **`status --require`は、書き忘れたファイルを見つけない。** fnaze は git を読まないので、何が変わったかを知らない。手順1の一覧と、`fnaze show --lang en`・`--lang ja`を見比べる
- **`rm`で子が無くなった骨組みの行も消えるので、あとから`set`した同じディレクトリは末尾に足される**（エントリの並びが前のファイルと変わる。仕様どおり）。並びを保ちたければ、`rm`の前に`set`する。2つの言語で、`set`・`rm`を同じ順に打つ
- **`fnaze new`を言語なしで打つと、いちばん新しい組と同じ言語で作る。** 前の組が`ja`だけなら`ja`だけになる。`en ja`を並べる
- **手元の`fnaze`が古いことがある。** `postCreate.sh`は、コンテナを作ったときのソースで入れる。fnaze のコマンドや形式を変えたコミットでは、先に`go install ./cmd/fnaze`で入れ直してから説明を書く（pre-commit も、PATH の`fnaze`を使う）
