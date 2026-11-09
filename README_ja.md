*[English](README.md) | **日本語***

# fnaze

> 英語版が正本（`README.md`）。この文書はその日本語版。

fnaze は、説明のファイルを作り、読み、検査する。説明のファイルには、変更のまとまり1つについて、変えたファイルごとに「何を、なぜ変えたか」を1行で書く。レビューする人は、この一覧と diff を並べて読む。

- 説明のファイル1つが、変更のまとまり1つを説明する。何を1つとするか（ふつうはコミット1つ）は使う人が決める
- fnaze は git を読まず、呼ばない。ほかのプログラムも呼ばない
- 説明は、いくつかの言語で持てる。言語ごとにファイル1つ

まだ実装中。次を見てほしい：

- [docs/reference/file-format_ja.md](docs/reference/file-format_ja.md)：ファイルの置き場所と書き方
- [docs/reference/commands_ja.md](docs/reference/commands_ja.md)：コマンド
- [docs/examples/](docs/examples/README_ja.md)：例のファイルと、git の pre-commit のフック

## ライセンス

MIT
