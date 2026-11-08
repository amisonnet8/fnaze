# 配布方法

<!-- TODO: このプロジェクトの配布方法（go install の有無、配る OS とアーキテクチャ、版の見せ方など）を、決まったら書く -->

## 配布物の作り方（goreleaser）

実行ファイルを配るときは、テンプレートに入っている`.goreleaser.yaml`・`.github/workflows/release.yml`・`tools/licenses`を使う。見本は crube（https://github.com/amisonnet8/crube ）。

- **始めるとき**：`cmd/<名前>/`に実行ファイルを置いたら、`.goreleaser.yaml`の`yourproject`（`project_name`・`main`・`binary`と、`tools/licenses`に渡す main パッケージ）を書き換え、`tools/licenses`の`TestRun`を、その実行ファイルのライセンスを集めるテストに書き換える（テストのコメント）。それまでは`qsoku snapshot`とリリースは、作るものが無く失敗する（`goreleaser check`は、GitHub の remote があるリポジトリなら通る）
- **cgo を使わない**（`CGO_ENABLED=0`）。どの OS でも、ほかのライブラリなしに動く
- 配布物は linux・darwin・windows の amd64・arm64。linux・darwin は`.tar.gz`、windows は`.zip`。名前は`<名前>_<版>_<OS>_<アーキテクチャ>`（版は`v`なし）。`checksums.txt`を添える。アーカイブには、実行ファイル・`LICENSE`・`README.md`・`licenses/`（依存のライセンス文）を入れる
- **版**：goreleaser が`-X main.version=<タグ>`（`v0.1.0`の形）を埋める。`main`パッケージに`var version string`を置き、空のとき（`go install`で入れたもの）は`runtime/debug.ReadBuildInfo`のモジュールの版を出すと、どちらでも同じ`v0.1.0`になる（crube の`cli.BuildVersion`）
- **依存のライセンス文**を配布物に同梱する。`tools/licenses`が、3つの OS で実行ファイルに組み込まれるモジュールを`go list -deps`で求め、そのライセンスのファイルと Go の標準ライブラリの`LICENSE`を`bin/licenses/<モジュールのパス>/`に写す（goreleaser の`before`の手順）。ライセンスのファイルが無いモジュールがあれば失敗し、リリースが止まる
  - 依存を足す・外すと、`tools/licenses`の`TestRun`の一覧が落ちる（書き換えたあと）。新しいモジュールのライセンスを確かめてから一覧を直す（Trivy の`qsoku trivy`も通す）

## リリースの流れ

ビルド済みのバイナリなどを GitHub Releases で配るときは、次の流れにする。どのプロジェクトも同じ。

1. **人間が**タグ（`v0.1.0`の形）を打って push する。人間がするのはこれだけ
2. タグで動く workflow（`.github/workflows/release.yml`）が、検査を通してから配布物を作り、Release に添付して**公開する**（本文は goreleaser が前のタグからのコミットで作る）

- **Claude Codeは、タグも Release も作らない。** `gh release create`は`.claude/settings.json`で拒否（deny）されている。Release を作るのは workflow の中だけ
- `release.yml`は、まず`ci.yml`をそのまま呼んで（`workflow_call`）普段の CI の検査をすべて通し、そのあと goreleaser（`.goreleaser.yaml`。`goreleaser-action`の`version: "~> v2"`で、v2 の最新）で配布物を作る。書き込みの権限（`contents: write`）は、Release を作るジョブだけに与える
- goreleaser はいったん**下書き**に添付し、すべて添付し終えてから、最後の手順（`gh release edit --draft=false`）で公開する。途中で落ちると下書きだけが残り、公開された Release に配布物が欠けることはない。同じタグでもう一度動かすと、その下書きを置き換える（`replace_existing_draft`）。落ちたときは、原因を直したコミットに新しいタグを打つか、同じコミットのままなら workflow を再実行する
- **公開済みの Release があるタグでは、何も変えずに失敗する**（最初の手順）。出し直すときは、新しいタグを打つ
- **タグを打つ前に、手元で`qsoku snapshot`を動かして中身を見る**。タグなしで`dist/`に同じ配布物を作る（版は`0.0.1-snapshot`、バイナリの版は`v0.0.0`）。goreleaser は`.devcontainer/postCreate.sh`が入れる。コンテナを作り直す前に要るときは、`GOTOOLCHAIN=auto go install github.com/goreleaser/goreleaser/v2@latest`（goreleaser がコンテナより新しい Go を求めるとき、その Go を取ってきてビルドする。Bashサンドボックスの中では、`proxy.golang.org`と`storage.googleapis.com`への接続が要る）。`.goreleaser.yaml`を変えたら`qsoku snapshot`で確かめる
- ビルド済みのバイナリは、リポジトリにコミットしない（`.claude/rules/directory-structure.md`「配布物」）

## VS Code の拡張を配るとき

VS Code の拡張（`.vsix`）を持つプロジェクトも、同じ流れにする。

- **拡張の版（`package.json`の`version`）は、タグから`v`を除いたものと同じにする**。Go のコマンドも配るプロジェクトでは、コマンドと拡張を同じ版で出す（Release に版の違う`.vsix`が並ぶと、どれが対か分からない）。拡張が変わらない版でも上げる。タグを打つ前に、`package.json`と`package-lock.json`（2か所）を次の版にしてコミットする
- `release.yml`の**最初のジョブ**で、タグと`package.json`の版を比べ、違えば何も作らずに止める（長い検査の前に。`jq -r .version`。見本は srwr）

- **Go のコマンドも配るプロジェクト**（goreleaser を使う）：`.vsix`は goreleaser の`before`の手順で作り（`bin/`の下へ）、`.goreleaser.yaml`の`release.extra_files`で同じ下書きに添付し、`checksum.extra_files`で`checksums.txt`にも入れる。`release.yml`は template のまま（`before`の手順が使う道具を入れる手順だけ足す）。`qsoku snapshot`でも`.vsix`ができる。見本は srwr
- **拡張だけのプロジェクト**（goreleaser を使わない）：`release.yml`で、公開済みならば失敗する最初の手順と`ci.yml`の検査のあと、`.vsix`を作り、`gh release create --draft`で下書きを作って添付し（本文は`CHANGELOG.md`のその版の節）、最後に公開する
- **Marketplace への公開は、人間が手で行う**（Marketplace の Web から`.vsix`を上げる。トークンを用意していないので、workflow からは公開しない）。GitHub Release に添付された`.vsix`を使う
