---
description: "AIチームでGitHub PRをレビュー: ghで差分取得 → Codex MCP & Gemini MCPに依頼 → Claude Codeが統合"
argument-hint: "<pr_url_or_number> [--repo <owner/repo>]"
allowed-tools: Bash(gh pr view:*), Bash(gh pr diff:*), Bash(gh pr checks:*), Bash(gh pr review:*), Bash(gh pr comment:*), Python(~/scripts/post-review-command.py)
---

# 実行順序（TL;DR）

1. **事前チェック**

- `gh auth status`（認証）
- `claude mcp get codex`（Codex MCP 接続）
- `claude mcp get gemini`（Gemini MCP 接続）
- `gh repo view [--repo]`（アクセス権）

2. **対象リポジトリの特定**（`--repo` 明示 or カレントから自動検出）
3. **PR コンテキスト取得**（view → diff --name-only → diff → checks）
4. **Claude 一次レビュー**（正当性/セキュリティ/性能/設計/テスト/ドキュメント）
5. **MCP 実行**（Codex & Gemini に同一コンテキスト投入 → 合意形成）
6. **所見統合・優先度付け**（Blocking / Should Fix / Nits / Open Questions）
7. **投稿方式の選択**（`--request-changes` / コメント / インライン）
8. **レビューコメント投稿**（`post-review-command.py`）
9. **投稿結果の確認**（GitHub 表示確認・リトライ）
10. **完了判定**（分析だけは不可。投稿＋確認まで）

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

### 0-2. MCP サーバー接続状況確認（Codex / Gemini）

```bash
claude mcp list
claude mcp get codex
claude mcp get gemini
```

### 0-3. 対象リポジトリアクセス権確認

```bash
# リポジトリが指定されている場合
gh repo view $REPO_URL

# 指定がない場合、カレントから自動検出
gh repo view
```

### 0-4. MCP ツール検出（動的）

- "review", "pr", "analyze" を含むツール名を探索。見つからなければ、Task tool へフォールバック。

> **補足（自動チェック不要だが重要）**
>
> - `gh` ログイン済みで PR にアクセス可能
> - Codex/Gemini MCP 接続済み（未接続でも Claude 単独で継続可）
> - `~/scripts/post-review-command.py` 実行可能

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

## フェーズ 3: Claude によるレビュー

1. **前提条件チェック**（フェーズ 0 の実施・対処）
2. **リポジトリ特定**（フェーズ 1）
3. **要約**：PR の目的・影響範囲・変更点（主要ファイル/ディレクトリ、追加/削除/変更の傾向）
4. **影響範囲の調査**：差分から影響が及ぶ領域を列挙（後続レビューのコンテキストに使用）
5. **Claude 一次レビュー**（具体指摘）

   - 正当性（仕様充足、境界条件、失敗系）
   - セキュリティ（入力検証、権限、シークレット、依存性）
   - パフォーマンス（計算量/メモリ、I/O、N+1、キャッシュ）
   - 可読性/設計（命名、分割、責務、凝集度/結合度）
   - テスト/検証（不足テスト、再現手順、フェイルファスト）
   - ドキュメント（README/ADR/コメント/マイグレーション手順）

---

## フェーズ 4：MCP 連携（Codex MCP & Gemini MCP）

- **動的ツール特定**：`review` / `pr` を含む最適ツールを特定
- **同一コンテキスト投入**：メタデータ+影響範囲がある実装の共有+PR で発生する差分+CI 要約
- **短いラリーで認識合わせ**：

  - それぞれの所見（Codex / Gemini）を回収
  - **同意点/相違点**を明示し、**1 往復以上**で合意形成

---

## フェーズ 5：統合・優先度付け（実行）

- **Blocking（必須修正）**：重大不具合/セキュリティ/仕様逸脱
- **Should Fix（推奨修正）**：品質/性能/保守性の明確改善
- **Nits（任意改善）**：表記・スタイル・微細最適化
- **Open Questions（確認事項）**：決定待ち/追加情報依頼

---

## フェーズ 6：レビューコメント投稿（実行）

> **投稿方式の選択**
>
> - Blocking / Should Fix あり → インライン（単発 or 一括）
> - Nits のみ → 通常コメント
> - 行単位の指摘 → インライン（単発 or 一括）

```bash
# 通常レビューコメント
python3 ~/scripts/post-review-command.py review "$ARGUMENTS" --request-changes --body "レビュー内容"

# 通常コメント
python3 ~/scripts/post-review-command.py comment "$ARGUMENTS" --body "コメント内容"

# インラインコメント
python3 ~/scripts/post-review-command.py inline "$ARGUMENTS" --file "ファイルパス" --line 行番号 --body "インラインコメント"

# インライン一括
python3 ~/scripts/post-review-command.py inline-batch "$ARGUMENTS" --json comments.json
```

**注意事項**

- `post-review-command.py` は対象リポジトリを自動検出し、適切な PR を対象にします。
- コメントは日英併記で投稿すること。
- Claude からの所感、Codex からの所感、Gemini からの所感をまとめ投稿に記載すること

---

## フェーズ 7：投稿結果確認 & 完了判定（実行）

**完了条件（すべて満たす）**

1. **レビュー結果の投稿完了**：`post-review-command.py` で投稿、失敗時は再試行
2. **投稿内容の確認**：GitHub 上に正しく投稿できているか（インラインコメントの投稿位置も含む）を、gh コマンドで確認する。
3. **投稿方法の妥当性**：

   - Blocking/Should Fix あり → インライン（単発 or 一括）
   - Nits のみ → 通常コメント
   - インライン必要箇所 → ファイル/行を指定

> **注意**：レビュー分析のみでは**未完了**。投稿と確認まで必須。

---

## エラーハンドリング（横断）

- **MCP 接続失敗**：Codex/Gemini 未接続を明記 → Claude 単独で継続可
- **GitHub API エラー**：自分の PR は自動 comment モード／レート制限はリトライ
- **リポジトリ検出失敗**：`--repo` を明示指示

---

## 実行フロー（段階まとめ）

1. 事前チェック → 2) リポジトリ確定 → 3) PR 情報取得 → 4) Claude 一次 → 5) MCP 実行 → 6) 統合 → 7) 投稿方式選択 → 8) 投稿 → 9) 確認 → 10) 完了

---

## 期待する出力フォーマット

- **前提条件チェック結果**（各チェックの結果と対処）
- **リポジトリ情報**（対象の特定結果）
- **Summary（要約）**（1–3 段落）
- **Blocking / Should Fix / Nits / Open Questions**（箇条書き。可能ならファイル/行を明記）
- **Agent Notes**（Codex 抜粋 / Gemini 抜粋 / 合意点・相違点・最終判断）
- **Proposed Commands**（`post-review-command.py` 実行例と必要に応じた JSON 雛形）
- **投稿完了の確認**（投稿結果と GitHub 上の表示確認）

---

## 品質基準

- **事実**（差分/メタ/CI）と**推測**を区別（推測は明記）
- 重要条件・数値は列挙
- 指摘は**具体・検証可能**・**再現手順付き**
- 大規模差分は**優先度付け**＋**代表例**
- **実行コマンド**提示と**インライン用 JSON**用意可
- **分析で終わらせない**：**投稿・確認まで**

---

## 使用例

```bash
# 現在リポジトリの PR を番号指定
python3 ~/scripts/pr-review-by-ai-team.py 123

# PR URL 指定
python3 ~/scripts/pr-review-by-ai-team.py https://github.com/owner/repo/pull/123

# 特定リポジトリを明示
python3 ~/scripts/pr-review-by-ai-team.py 123 --repo owner/repo
```

### 通常レビューコメント（Request Changes）

```bash
python3 ~/scripts/post-review-command.py review "$ARGUMENTS" --request-changes --body "レビュー内容"
```

### インラインコメント一括投稿（サンプル JSON）

```json
{
  "event": "COMMENT",
  "body": "以下の指摘事項を確認してください",
  "comments": [
    {
      "path": "src/main.py",
      "line": 42,
      "side": "RIGHT",
      "body": "この関数はエラーハンドリングが不足しています"
    },
    {
      "path": "src/main.py",
      "line": 45,
      "start_line": 43,
      "side": "RIGHT",
      "body": "この範囲のコードは重複しているため、共通関数に抽出してください"
    }
  ]
}
```

```bash
python3 ~/scripts/post-review-command.py inline-batch "$ARGUMENTS" --json comments.json
```

### サンプルファイルの使用

```bash
cp ~/scripts/sample-comments.json my-review.json
python3 ~/scripts/post-review-command.py inline-batch "$ARGUMENTS" --json my-review.json
```
