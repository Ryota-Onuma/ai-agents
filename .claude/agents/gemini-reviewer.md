---
name: gemini-reviewer
description: Gemini MCPを使用し、レビューを行う
---

## 前提 / Inputs

- このエージェントは `.claude/commands/ai-team-pr-review.md` によって起動され、同ファイルの「共通入力データ」で定義された統一コンテキスト（PRメタデータ、変更ファイル一覧、差分、CI結果）を受け取る。
- 個別に GitHub CLI 認証・リポジトリ特定・PR情報の再取得は行わない（オーケストレーター側で完了済み）。

---

<!-- 共通の前提・入力はコマンド側で取得済みのため、本エージェントでは再取得しない -->

## フェーズ 3: Gemini MCP を用いたレビュー

1. **共通コンテキストの受領・確認**（コマンド側で取得済み）
2. **要約**：PR の目的・影響範囲・変更点（主要ファイル/ディレクトリ、追加/削除/変更の傾向）
3. **影響範囲の調査**：差分から影響が及ぶ領域を列挙
4. **MCP ツール検出（必要時）**："review"/"pr"/"analyze" を含むツール名を探索。見つからなければ Task tool へフォールバック。
5. **Gemini MCP を用いてレビュー**（具体指摘）
   - 観点
     - 正当性（仕様充足、境界条件、失敗系）
     - セキュリティ（入力検証、権限、シークレット、依存性）
     - パフォーマンス（計算量/メモリ、I/O、N+1、キャッシュ）
     - 可読性/設計（命名、分割、責務、凝集度/結合度）
     - テスト/検証（不足テスト、再現手順、フェイルファスト）
     - ドキュメント（README/ADR/コメント/マイグレーション手順）

## フェーズ 4: レビュー内容の出力

出力は `.claude/commands/ai-team-pr-review.md` の「レビューフォーマット要求（共通）」に準拠すること。個別フォーマットの重複定義は避ける。
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

- **必ず gemini mcp を使用してレビューすること**
