# GitHub PR Review Command Tool - 改良版

## 概要

`post-review-command-improved.py` は、GitHub PR にレビューコメントやインラインコメントを投稿するための改良版 Python スクリプトです。

## 主な改良点

### 1. 一時ファイルを使用した長文コンテンツ対応

- 長いレビュー内容をコマンドライン引数で渡す際のエスケープ問題を解決
- 一時ファイルを使用して安全にコンテンツを投稿

### 2. PR 作成者の自動判定

- 自分の PR の場合は自動的に制限を回避
- `--request-changes` が使用できない場合の自動フォールバック

### 3. エラーハンドリングの強化

- レート制限対応（指数バックオフ付きリトライ）
- より詳細なエラーメッセージと対処方法の提示

### 4. 堅牢性の向上

- 例外処理の強化
- ユーザー中断時の適切な処理

## 使用方法

### 基本的な使い方

```bash
# 通常レビューコメント
python3 post-review-command-improved.py review <PR> --body "レビュー内容"

# 通常コメント
python3 post-review-command-improved.py comment <PR> --body "コメント内容"

# インラインコメント
python3 post-review-command-improved.py inline <PR> --file "ファイルパス" --line 行番号 --body "インラインコメント"

# インラインコメント一括投稿
python3 post-review-command-improved.py inline-batch <PR> --json comments.json
```

### PR 指定方法

PR は以下のいずれの形式でも指定可能です：

- PR 番号（例：`123`）
- PR URL（例：`https://github.com/owner/repo/pull/123`）
- ブランチ名（例：`feature/new-feature`）

### レビュータイプの指定

```bash
# Approve
python3 post-review-command-improved.py review <PR> --approve --body "承認します"

# Request Changes
python3 post-review-command-improved.py review <PR> --request-changes --body "修正が必要です"

# Comment（デフォルト）
python3 post-review-command-improved.py review <PR> --comment --body "コメントです"
```

**注意**: 自分の PR に対して `--request-changes` を使用した場合、自動的に通常のコメントとして投稿されます。

## インラインコメント一括投稿

### JSON ファイルの形式

```json
{
  "event": "COMMENT",
  "body": "レビューサマリー",
  "comments": [
    {
      "path": "ファイルパス",
      "line": 行番号,
      "side": "RIGHT",
      "body": "コメント内容"
    },
    {
      "path": "ファイルパス",
      "line": 最終行,
      "start_line": 開始行,
      "side": "RIGHT",
      "body": "複数行コメント内容"
    }
  ]
}
```

### サンプルファイル

`sample-comments.json` ファイルを参考にしてください。

## エラーハンドリング

### レート制限対応

- 自動的にリトライを実行
- 指数バックオフによる待機時間の調整

### 自分の PR の制限回避

- 自動的に PR 作成者を判定
- 制限のある操作を適切な代替手段に変更

### エラー時の対処

- 詳細なエラーメッセージの表示
- 対処方法の提案

## 前提条件

- Python 3.6 以上
- GitHub CLI (`gh`) がインストール済み
- GitHub CLI にログイン済み（`gh auth login`）
- 対象リポジトリへのアクセス権限

## トラブルシューティング

### よくある問題

1. **gh CLI が見つからない**

   - `gh` コマンドがインストールされているか確認
   - PATH に正しく設定されているか確認

2. **認証エラー**

   - `gh auth status` でログイン状態を確認
   - 必要に応じて `gh auth login` を実行

3. **権限エラー**

   - 対象リポジトリへのアクセス権限を確認
   - 組織の設定で PR への書き込み権限があるか確認

4. **レート制限エラー**
   - 自動的にリトライされます
   - 長時間待機が必要な場合は手動で再実行

## 更新履歴

- **v2.0.0**: 改良版リリース
  - 一時ファイル対応
  - PR 作成者自動判定
  - エラーハンドリング強化
  - レート制限対応

## ライセンス

このスクリプトは MIT ライセンスの下で提供されています。
