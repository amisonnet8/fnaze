# ディレクトリ構成

新しいファイルを追加する際は、どこに置くかをここで確かめる。**構成を変えたら、この文書も同じ変更の中で直す。**

**このリポジトリの説明（`.crube/`）も保つ。** ファイル・ディレクトリを足す・動かす・消すときは、同じ変更の中で英語と日本語の説明を直し、コミットの前に両方の言語で`crube status --require all`を通す。やり方（MCPとCLI、`.crube/ignore`の保ち方）は`.claude/rules/crube-usage.md`

```
.
├── CLAUDE.md               ← プロジェクトルール（参照先の案内）
├── LICENSE                 （MIT）
├── README.md
├── qsokufile               ← ビルド・テストの近道（qsoku）
├── go.mod / go.sum
├── .gitattributes          （`* text=auto eol=lf`）
├── .gitignore
├── .golangci.yaml          ← lintの設定
├── .goreleaser.yaml        ← 配布物の作り方（goreleaser。`release.yml`と`qsoku snapshot`が使う。`.claude/rules/distribution.md`）
├── trivy.yaml              ← 脆弱性・ライセンス検査の設定
├── .mcp.json               ← mtqg・crube（英語の`crube`・日本語の`crube-ja`）・fnazeのMCPサーバーの配線
├── .crube/                 ← このリポジトリのファイルの説明（crube。`structure.txt`は英語、`structure_ja.txt`は日本語、`ignore`は`.gitignore`に合わせて手で保つ）。**コミットする**
├── .fnaze/changes/         ← このリポジトリのコミットごとの説明（fnaze 自身で書く。`.claude/rules/fnaze-usage.md`）。**コミットする**
├── .githooks/              ← git のフック（`pre-commit`は`docs/examples/pre-commit`と同じ中身。`core.hooksPath`は`postCreate.sh`が設定する）
├── cmd/fnaze/              ← 実行ファイル fnaze の main（引数を internal/cli に渡すだけ）
├── internal/               ← このモジュールの中だけで使うコード（cli がコマンドライン、mcpserver が MCP サーバー、core がすべての判定）
├── e2e/                    ← ビルドした fnaze を動かすテストと、docs の例を実際の出力と突き合わせるテスト（`go test ./...`で動く。`.claude/rules/testing.md`「e2e」）
├── tools/licenses/         ← 配布物に入れる依存のライセンス文を集める道具（goreleaser が`go run`で動かす。実行ファイルには組み込まれない）
├── docs/                   ← 文書
├── .devcontainer/          ← 開発環境（devcontainer.json、postCreate.sh）
├── .claude/                ← rules/（ルール）、skills/（mutation-check：壊して確かめる手順と、変異の道具`mutants/`（別の Go モジュール）・仕様の変異の一覧`spec.txt`）、hooks/（build.sh、crube-untracked.sh：Write のあとに未説明を知らせる）、settings.json（人間が管理する）
├── .github/workflows/      ← CI（`ci.yml`）と、タグで動くリリース（`release.yml`。`ci.yml`を呼んでから Release を作って公開する）
└── .mtqg/                  ← mtqgの記録
```

## 配置の判断基準

<!-- TODO: このプロジェクトのコードの層・パッケージの分け方と、「どのコードをどこに置くか」の判断基準を書く -->

## 各ファイル・ディレクトリの補足

- **`cmd/`**: 実行ファイル`fnaze`を`cmd/fnaze/`に置く。`cmd/fnaze/main.go`は引数を`internal/cli`に渡すだけにする
- **`tools/licenses/`**: 配布物に同梱する依存のライセンス文を集める（`.claude/rules/distribution.md`）。`go run ./tools/licenses <出力先> <main パッケージ>`。実行ファイルを置いたら、`TestRun`をその実行ファイルの一覧のテストに書き換える
- **`.claude/skills/mutation-check/mutants/`**: 変異の道具。`go.mod`を持つ別のモジュールなので、`qsoku check`（`./...`）の対象に入らない。変えたら、そのディレクトリで整形・vet・テストを通す（Skill の「気をつけること」）
- **`internal/`**: Goの仕組みとして、モジュールの外からimportできない。外へ公開するパッケージは`internal/`に置かない
- **`go.sum`**: コミットする（改ざん検知・再現可能なビルドのため）
- **`.mtqg/`**: コミットされる。中のファイルは直接編集せず、すべて`mtqg`のコマンド経由で書く（`.claude/rules/mtqg-usage.md`）。`.mtqg/.local/`はコミットされない
- **`.mcp.json`**: `mtqg mcp`（MCPサーバー）の配線。`mtqg init --agent claude-code`が書くものと同じ。その後ろに、crubeのMCPサーバー（`crube mcp --allow-write`。`crube-ja`は`CRUBE_LANG=ja`）を手で足している（`.claude/rules/crube-usage.md`）。最後に、fnaze 自身のMCPサーバー（`fnaze mcp --allow-write`。1つだけ）を足している（`.claude/rules/fnaze-usage.md`）
- **`.crube/`**: このリポジトリのファイルとディレクトリの説明（`.claude/rules/crube-usage.md`）。**コミットする**（`.gitignore`の`!/.crube/`）。追跡するファイルすべての説明を、両方の言語で持つ。`ignore`は`.gitignore`を手で写したもの（crube は`.gitignore`を読まない）
- **`.devcontainer/devcontainer-lock.json`**: コンテナのビルドで自動で作られる。**コミットする**（この版のfeatureでテストなどが通った証明になる）。featureを新しくしたいときは、lockを消してビルドし直し、新しいlockをコミットする
- **VSCodeの設定**: `.devcontainer/devcontainer.json`の`customizations.vscode`に置く（拡張と設定を1か所にまとめる）。`.vscode/`は作らない
- **`.gitattributes`**: `* text=auto eol=lf`。Windowsランナーでの改行コード変換による誤検知を防ぐ（`.claude/rules/testing.md`）。`.mtqg/`の中の`.gitattributes`は`mtqg init`が置くもので、別物
- **`.gitignore`のドットファイル**: ルート直下のドットファイル・ドットディレクトリは**許可リスト方式**（`/.*`を無視し、追跡するものだけ`!`で戻す。`.claude/`の中は`/.claude/*`を無視して`settings.json`・`rules/`・`skills/`・`hooks/`だけ戻す）。サンドボックスが作業ディレクトリ直下に出すダミーファイルを`git add -A`で混ぜないため（`.claude/rules/testing.md`）。**追跡するドットファイルを新しく足すときは、`.gitignore`に`!`の行を足すこと**。足さないと`git add`が無視する
- **配布物（ビルド済みバイナリ）**: リポジトリにコミットしない。`.gitignore`には**ルート直下に限定して**書く（`/名前`）。名前だけを書くと、同名のソースディレクトリまで無視される
- **`docs/`**: 文書。`design/`（設計）、`reference/`（仕様）、`examples/`（使う人のための例）。どれも英語が正本で、`*_ja.md`が日本語版（`README.md`と`README_ja.md`も同じ）。言語の決まりは`CLAUDE.md`「ドキュメントの言語」
- **`.fnaze/`**: コミットごとの説明を、fnaze 自身のコマンドで書き、そのコミットに入れる（`.claude/rules/fnaze-usage.md`）。**コミットする**（`.gitignore`と`.crube/ignore`の`!/.fnaze/`）
- **`.githooks/`**: 追跡する git のフック。`.git/hooks/`は git に乗らないので、ここに置いて`core.hooksPath`で指す（`.devcontainer/postCreate.sh`）。`pre-commit`は`docs/examples/pre-commit`と同じ中身にし、e2e の`TestOwnPreCommit`が同じことを確かめる（例を直したら、こちらも写す）
- **シェルスクリプト**: 最小限にする。追跡中のものは`qsoku shellcheck`の対象になる（`.claude/rules/testing.md`）
