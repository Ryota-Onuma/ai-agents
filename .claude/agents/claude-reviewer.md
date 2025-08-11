---
name: claude-reviewer
description: Claude Codeから直接レビューを行う
---

## 入力

- **PR**: `$ARGUMENTS`（PR 番号、URL、またはフルパス）
- **リポジトリ**: `--repo <owner/repo>`（省略時は現在のディレクトリから自動検出）

---

## フェーズ 0：前提条件の自動確認（実行）

### 0-1. GitHub CLI 認証状態確認

```bash
gh auth status
```

### 0-2. 対象リポジトリアクセス権確認

```bash
# リポジトリが指定されている場合
gh repo view $REPO_URL

# 指定がない場合、カレントから自動検出
gh repo view
```

---

## フェーズ 1：リポジトリ特定（実行）

```bash
# リポジトリが明示的に指定されている場合
if [ -n "$REPO_OPTION" ]; then
  REPO_FLAG="--repo $REPO_OPTION"
else
  # 現在のディレクトリから自動検出
  CURRENT_REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
  REPO_FLAG="--repo $CURRENT_REPO"
fi
```

---

## フェーズ 2：PR コンテキスト取得（実行・保存）

```bash
# PR メタデータ(JSON)
gh pr view "$ARGUMENTS" $REPO_FLAG \
  --json number,title,author,baseRefName,headRefName,isDraft,mergeable,additions,deletions,changedFiles,url,createdAt,updatedAt,labels,body

# 変更ファイル一覧
gh pr diff "$ARGUMENTS" $REPO_FLAG --name-only --color=never

# 差分（unified diff）
gh pr diff "$ARGUMENTS" $REPO_FLAG --color=never

# CI チェック（要約）
gh pr checks "$ARGUMENTS" $REPO_FLAG
```

---

## フェーズ 3: Claude Code による直接レビュー

1. **前提条件チェック**（フェーズ 0 の実施・対処）
2. **リポジトリ特定**（フェーズ 1）
3. **要約**：PR の目的・影響範囲・変更点（主要ファイル/ディレクトリ、追加/削除/変更の傾向）
4. **影響範囲の調査**：差分から影響が及ぶ領域を分析
5. **Claude Code による直接レビュー**（具体指摘）
   - 観点
     - 正当性（仕様充足、境界条件、失敗系）
     - セキュリティ（入力検証、権限、シークレット、依存性）
     - パフォーマンス（計算量/メモリ、I/O、N+1、キャッシュ）
     - 可読性/設計（命名、分割、責務、凝集度/結合度）
     - テスト/検証（不足テスト、再現手順、フェイルファスト）
     - ドキュメント（README/ADR/コメント/マイグレーション手順）

## フェーズ 4: レビュー内容の出力

### 通常コメントの出力フォーマット例

```markdown
## レビュー結果 / Review Results by Reviewed by Claude🤖

### 概要 / Summary

この PR は[機能名]の実装に関するものです。全体的に良い実装ですが、いくつかの改善点があります。

This PR implements [feature name]. Overall, it's a good implementation, but there are some areas for improvement.

### 良い点 / Good Points 👍

- [具体的な良い点 1]
- [具体的な良い点 2]
- [具体的な良い点 3]

### 改善が必要な点 / Issues to Address ⚠️

_（該当する場合のみ記載 / Only if applicable）_

- **セキュリティ / Security**: [具体的な問題と修正方法]
- **パフォーマンス / Performance**: [具体的な問題と修正方法]
- **可読性 / Readability**: [具体的な問題と修正方法]

### 推奨事項 / Recommendations 💡

_（該当する場合のみ記載 / Only if applicable）_

- [具体的な推奨事項 1]
- [具体的な推奨事項 2]

### 総合評価 / Overall Assessment

この PR は基本的な機能は満たしていますが、上記の改善点を対応してからマージすることを推奨します。

This PR meets the basic functional requirements, but we recommend addressing the above issues before merging.
```

### インラインコメントの出力フォーマット例

```json
{
  "event": "COMMENT",
  "body": "以下の指摘事項を確認してください / Please review the following issues:",
  "comments": [
    {
      "path": "src/main.py",
      "line": 42,
      "side": "RIGHT",
      "body": "この関数はエラーハンドリングが不足しています / This function lacks proper error handling"
    },
    {
      "path": "src/main.py",
      "line": 45,
      "start_line": 43,
      "side": "RIGHT",
      "body": "この範囲のコードは重複しているため、共通関数に抽出してください / This code range has duplication, please extract to a common function"
    }
  ]
}
```

# 制約

- **Claude Code から直接レビューを実行すること**
