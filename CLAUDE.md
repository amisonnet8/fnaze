# CLAUDE.md

このファイルは、このプロジェクトで作業する際、Claude Codeが常に踏まえるべき前提を示す。

## このプロジェクトについて

fnaze は、変更のまとまりごとに、変えたファイルそれぞれについて「何を、なぜ変えたか」を1行で書いた説明のファイルを作り、読み、検査する CLI。人間は、この一覧と diff を並べて見ればレビューできる。説明のファイル1つが、変更のまとまり1つ（一般的な使い方はコミット1つにつき1つ）。

- 設計：`docs/design/overview.md`
- ファイルの形式：`docs/reference/file-format.md`（日本語版 `file-format_ja.md`）
- コマンド：`docs/reference/commands.md`（日本語版 `commands_ja.md`）
- 例：`docs/examples/`

## 守ること（設計の芯）

実装の細部より優先する。これを崩す変更は、実装する前に必ず相談すること。

- **git などのバージョン管理と関わらない。** fnaze は git を読まず、呼ばない。ほかのプログラムも呼ばない。検査は説明のファイルの中だけで閉じる
- **`.fnaze/` の中のほかは書かない**
- **判定はコアに集め、CLI と MCP はその薄いラッパーにする**：両者の間に機能差・挙動のずれを出さない。MCP の道具は、CLI に `--json` を付けたときと同じ JSON を返す
- **説明のファイル1つが、まとまり1つ。** それがどの変更を指すかは、fnaze は決めない。区切り方や検査を強いない（止めたい人には例を示すだけ）
- **ファイル名は `<日時>_<UUID>[_<言語の名前>].txt`。** 言語ごとのファイルは `fnaze new` が同時に作り、名前をそろえる
- **言語の名前は、ただの文字列。** fnaze は `ja` が日本語であることを知らない。特別なのは `en`（接尾辞を付けない）だけ
- **役割外の機能を足さない**：変えたファイルの一覧・変更の種類を求めること、diff・編集の操作・コマンド・設計の判断・todo を記録すること、プロジェクトの今の構成を説明すること

## ドキュメントの言語

- `CLAUDE.md`・`.claude/rules/`・`docs/design/`は日本語のみ
- `docs/reference/`・`docs/examples/` の README は英語が正本で、日本語版`*_ja.md`と2本立て
- `README.md`は英語のみ
- コード・コメント・コミットメッセージ・CLI出力は英語
- 詳しい書き方は`.claude/rules/documentation.md`

## 参照すべきファイル

作業を始める前に、以下を確認すること。

- **`.claude/rules/mtqg-usage.md`** — mtqgの使い方（記録者、種類の使い分け、こまめに記録する）
- **`.claude/rules/qsoku-usage.md`** — qsokuの使い方（`qsokufile`の名前の一覧、呼び方、書式、`//`、シェル連携）。**ビルド・テスト・lintは`go`などを直接打たず、ここの名前で実行する**
- **`.claude/rules/crube-usage.md`** — crubeの使い方（MCPの`crube`・`crube-ja`とCLI、場所の探し方、英語と日本語の説明の保ち方）。**場所を探す前、ファイルを足す・動かす・消す前に読む**
- **`.claude/rules/testing.md`** — テスト方針（開発環境の前提、動作確認の粒度、期待値の決め方、壊して確かめる（Skill`mutation-check`）、Trivy、ShellCheck、`-race`、Bashサンドボックス・CIの落とし穴）
- **`.claude/rules/directory-structure.md`** — ディレクトリ構成と、ファイルの置き場所の判断基準。**新しいファイルを追加するときに読む**
- **`.claude/rules/datetime.md`** — 日時の扱い（データは UTC で持ち、表示だけ利用者のタイムゾーンに直す。瞬間で比べる、時計を差し替える）。**日時を保存・表示・比較するコードを書くときに読む**
- **`.claude/rules/working-with-human.md`** — 人間とのやり取り（頼み方の3種類、小さなつまずきの頼み方、確認の自動化、見え方を変えるときの了承）。**人間に何かを頼む前に読む**
- **`.claude/rules/naming.md`** — 命名規則（製品名、用語の表（英語と CLI・JSON での表記）、コマンドと MCP の道具の名前）。**用語を使う・足すときに読む**
- **`.claude/rules/distribution.md`** — 配布方法（goreleaserでの配布物の作り方とリリースの流れ。そのほかは未記入で、決まったら書く）
- **`.claude/rules/documentation.md`** — ドキュメントの書き方（例の値は道具で求める、`docs/design/`に書く経緯の線引き。そのほかは未記入で、決まったら書く）

これらのルールファイルは、実装中に得た細かい気づき・教訓をClaude Code自身が育てていくものである。新しく気づいたルール・踏んだ落とし穴があれば、該当するファイルに追記すること。どのファイルにも当てはまらない新しい種類の気づきであれば、新しいルールファイルを作ってよい（作ったらこの一覧にも足すこと）。

## 文書と今のコードが食い違うとき

- **文書（`CLAUDE.md`・`.claude/rules/`・`docs/`・コメント）の書き方と、すでにあるコードの書き方が食い違うときは、コードの書き方に従う。** 命名・構成・エラーの扱い・テストの形などは、文書の例より、今のコードで実際に使われている形に合わせて書く
- **見つけた食い違いは、その作業の最後の返答で全部挙げる。** 文書のどこと、コードのどこが、どう違うかを1件ずつ書く。どちらに合わせて直すかは人間が決めるので、文書もコードも、その場では直さない（`.claude/rules/working-with-human.md` 1章「別の作業の流れで見つけた問題は、先に直さず提案する」）

## 開発の進め方

- **ファイルの変更は、組み込みの`Edit`・`Write`で行う。** Bashでの書き換え（`sed -i`・リダイレクト）はしない（ビルドとcrubeのフックが動かない。下の「権限・自動化について」）
- **このリポジトリの中の場所は、まずcrubeで引く**（MCPの`find`・`tree`・`show`、または`crube find …`）。ファイル・ディレクトリを足す・動かす・消すときは、同じ変更の中で英語と日本語の説明を直し、コミットの前に両方の言語で`crube status --require all`を通す（`.claude/rules/crube-usage.md`）
- 実装後は`.claude/rules/testing.md`に従い、ビルド確認に加えて実際の動作確認（`qsoku check`、必要に応じて`qsoku race`・`qsoku trivy`・`qsoku shellcheck`）を行うこと。判断に迷ったら実行する側に倒すこと
- 依存ライブラリは、標準ライブラリ → `golang.org/x` → 何かの公式のもの → デファクトの順で選ぶ（依存を足すこと自体が人間に聞く判断。`.claude/rules/working-with-human.md`）
- 依存ライブラリを足したら、Trivyでライセンス・脆弱性を確認すること（`qsoku trivy`）

## gitの扱い（Claude Codeの作業として）

- **pushは人間が行う。** Claude Codeはコミットまでで止めること。`git push`は`.claude/settings.json`で拒否（deny）されている
- 作業の区切りでこまめにコミットしてよい（消してしまったときに戻せるように）
- `git reset --hard`・`git clean`は、コミットしていない作業を消すので、使う前に必ず理由を説明すること
- **コミットの前に`git config user.email`／`user.name`を確かめる。** 正しい値は`65074207+amisonnet8@users.noreply.github.com`／`amisonnet8`（GitHubのプライバシー用noreplyアドレス）。違う値なら、`.git/config`の`[user]`（ローカルの上書き）を疑う。ローカルに設定が無いと、コンテナのglobalの設定（noreplyでないことがある）が使われる

## 権限・自動化について

`.claude/settings.json`（人間が管理する）により、次の設定になっている。

- 拒否（deny）：`git push`、`gh pr merge`、`gh release create`
- 実行前に確認（ask）：`git reset --hard`、`git clean`、`sudo`、`gh pr create`
- **Bashコマンドは既定でOSレベルのサンドボックス（Linux bubblewrap）内で実行される。** ファイルシステムの書き込み先とネットワーク接続先は許可リストで絞られている。この範囲内なら`curl`・`wget`・`go get`などは確認なしに実行できる。リストに無い宛先・書き込み先は`sandbox_violations`として拒否される。そのときは、`.claude/rules/working-with-human.md` 2章の形（原因と、`.claude/settings.json` のどこに何を足すか）で**短く頼む**（許可を広げるかは人間が判断する）。落とし穴は`.claude/rules/testing.md`「Bashサンドボックスの落とし穴」参照

確認を求められた場合、無理に実行しようとしない。`.claude/settings.json`を変更したいときは、変更案を差分で示すにとどめること（人間が貼るだけで済む形にする）。

ビルドの自動フック（`PostToolUse`）が設定されている。`Edit`・`Write`・`MultiEdit`で`.go`・`go.mod`・`go.sum`を変えると`.claude/hooks/build.sh`が`qsoku build`を実行し、失敗すると理由がClaude Codeに返る。複数ファイルにまたがる編集の途中は、まだ書いていないファイルを参照して一時的に失敗することがある。

**crubeの配線**：`.mcp.json`に、crubeのMCPサーバーが2つある（`crube`は英語、`crube-ja`は日本語。`crube mcp --allow-write`）。`Write`のあとには`.claude/hooks/crube-untracked.sh`が、説明の無いパスを言語ごとに知らせる（止めはしない）。詳細は`.claude/rules/crube-usage.md`。

## mtqgでの記録について

作業の中で生まれる「気づき・やること・質問と答え・不具合とそのやり取り・用語の合意・この先も効き続ける決まり事」は、コードにもコミットにも残らず、チャットの履歴やメモアプリに散って失われやすい。`mtqg`（`m`emo & rules・`t`odo・`q`a & bugs・`g`lossary）はそれを`.mtqg/journal.jsonl`に追記し、gitに乗せて共有する。使い方は`.claude/rules/mtqg-usage.md`。`.mtqg/`には、テンプレートから引き継いだ決まり事（rule）と、このPJで決めた決まり事・用語（glossary）・やること（todo）を登録済み——`mtqg context`で確認できる。**作業を始めるときは、`mtqg context`の開いている todo から取りかかる。**AIエージェントからは、フック（`.claude/settings.json`の`mtqg hook`）とMCPサーバー（`.mcp.json`の`mtqg mcp`）の両方で使える。

## ルール・スキルの提案

作業を進める中で、以下に気づいたら、都度こちらから提案すること（提案するだけで、勝手に作成・適用はしない。判断はこちらが行う）。

- **新しいルールにした方がよさそうな知見**: 同じ種類の判断や落とし穴に複数回遭遇した、既存の`.claude/rules/`のどれにも当てはまらない、といった場合。どのファイルに追記すべきか、あるいは新規ファイルが必要かも合わせて提案する
- **Skill化した方が効率的そうな作業**: 「同じ手順を3回目繰り返している」など、パターン化できそうな一連の作業に気づいた場合。まだ実装が薄い段階では無理に提案する必要はない。パターンが実際に繰り返されてから提案すること

提案は気づいたタイミングで随時行ってよく、まとめて報告するために貯めておく必要はない。

This repository records its development with mtqg: run `mtqg context` at the start of a session, and see `.mtqg/SCHEMA.md` for the data format.
