---
description: "並列AIチームでPRレビュー: Claude、Codex、Geminiエージェントを同時実行し統合レビューをGitHubに投稿"
argument-hint: "<pr_url_or_number> [--repo <owner/repo>]"
allowed-tools: Bash(gh pr view:*), Bash(gh pr diff:*), Bash(gh pr checks:*), Bash(gh pr review:*), Bash(gh pr comment:*), Task(*), Python(~/scripts/post-review-command.py)
---

# AI チーム並列 PR レビューコマンド

3つのAIエージェント（Claude、Codex MCP、Gemini MCP）を並列で実行し、統合されたレビューをGitHubに投稿します。

## 実行フロー

1. **前提条件チェック** (並列実行)
2. **PR情報取得**
3. **3エージェント並列レビュー** (Claude、Codex、Gemini)
4. **レビュー統合・優先度付け**
5. **GitHubに投稿**

---

## 入力

- **PR**: `$ARGUMENTS`（PR番号、URL、またはフルパス）
- **リポジトリ**: `--repo <owner/repo>`（省略時は現在のディレクトリから自動検出）

---

## フェーズ 0: 前提条件チェック（並列実行）

### 0-1. GitHub CLI認証確認
```bash
gh auth status
```

### 0-2. MCPサーバー接続確認
```bash
# 並列チェック
claude mcp get codex &
claude mcp get gemini &
wait
```

### 0-3. リポジトリアクセス権確認
```bash
# リポジトリが指定されている場合
if [ -n "$REPO_OPTION" ]; then
  gh repo view "$REPO_OPTION"
else
  gh repo view
fi
```

---

## フェーズ 1: PRコンテキスト取得

```bash
# 対象リポジトリの決定
if [ -n "$REPO_OPTION" ]; then
  REPO_FLAG="--repo $REPO_OPTION"
else
  CURRENT_REPO=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
  REPO_FLAG="--repo $CURRENT_REPO"
fi

# PR情報取得（並列実行）
gh pr view "$ARGUMENTS" $REPO_FLAG --json number,title,author,baseRefName,headRefName,isDraft,mergeable,additions,deletions,changedFiles,url,createdAt,updatedAt,labels,body &
gh pr diff "$ARGUMENTS" $REPO_FLAG --name-only --color=never &
gh pr diff "$ARGUMENTS" $REPO_FLAG --color=never &
gh pr checks "$ARGUMENTS" $REPO_FLAG &
wait
```

---

## フェーズ 2: AIエージェント並列レビュー

### Task toolを使用して3エージェントを並列実行

1. **claude-reviewer** エージェント
   - Claude Codeによる直接レビュー
   - 基本的な品質チェック（正当性、セキュリティ、パフォーマンス、可読性）

2. **codex-reviewer** エージェント
   - Codex MCPを使用したレビュー
   - GPT-5による高度なコード解析

3. **gemini-reviewer** エージェント
   - Gemini MCPを使用したレビュー
   - Geminiによる多角的分析

### 共通入力データ
各エージェントに以下の統一コンテキストを提供：
- PR メタデータ（JSON）
- 変更ファイル一覧
- 差分内容（unified diff）
- CIチェック結果

### エージェント共通指示事項
各エージェントには以下の形式でレビューを依頼：

**重要**: 具体的な問題を発見した場合は、必ず **ファイル名:行番号** を明記してください。

**レビューフォーマット要求**（日英併記必須）:
```markdown
## {エージェント名} Review

### 具体的指摘事項 / Specific Issues (インラインコメント対象 / Inline Comment Target)
#### 🚨 Blocking (Must Fix) / 必須修正
- **ファイル名:行番号**: 日本語での問題説明と修正提案
- **Filename:Line**: English problem description and fix suggestion

#### 💡 Should Fix (Recommended) / 推奨修正  
- **ファイル名:行番号**: 日本語での改善提案
- **Filename:Line**: English improvement suggestion

#### 🔧 Nits (Optional) / 任意改善
- **ファイル名:行番号**: 日本語での細かな改善点
- **Filename:Line**: English minor improvement

### 全体的所見 / Overall Insights (全体コメント対象 / General Comment Target)
[アーキテクチャ、設計思想、全体構造に関する所見]
[Insights about architecture, design philosophy, and overall structure]
```

---

## フェーズ 3: レビュー結果統合

### 3-1. 各エージェントの結果を収集
- Claude reviewer の所見
- Codex reviewer の所見  
- Gemini reviewer の所見

### 3-2. 統合分析実行
- **合意点の抽出**: 3つのエージェントが共通して指摘する問題
- **相違点の分析**: エージェント間で意見が分かれる箇所
- **優先度付け**:
  - **Blocking（必須修正）**: セキュリティ、重大バグ、仕様逸脱
  - **Should Fix（推奨修正）**: 品質、性能、保守性の改善
  - **Nits（任意改善）**: スタイル、微細最適化
  - **Open Questions（確認事項）**: 追加情報が必要な項目

### 3-3. 統合レビューコメント作成

#### 3-3-1. インラインコメント対象の特定
各エージェントのレビューから、具体的なファイル・行番号が特定できる指摘を抽出：

```bash
# 各エージェントレビューから行番号付き指摘を抽出
grep -n "Line [0-9]*:" /tmp/claude_review.md > /tmp/claude_inline_issues.txt
grep -n "行 [0-9]*:" /tmp/codex_review.md > /tmp/codex_inline_issues.txt
grep -n "content/posts/.*.md:[0-9]*" /tmp/gemini_review.md > /tmp/gemini_inline_issues.txt
```

#### 3-3-2. インラインコメントJSON生成
特定された問題を GitHub API 形式のJSONに変換：

- **優先度別コメント作成**:
  - 🚨 Blocking: セキュリティ、バグ、設定エラー
  - 💡 Should Fix: 品質改善、最適化、一貫性
  - 🔧 Nits: スタイル、タイポ、微調整

- **コメント内容構成**（日英併記必須）:
  ```
  {優先度アイコン} **{カテゴリ}**: {日本語での問題説明}
  {Priority Icon} **{Category}**: {English problem description}
  
  {修正提案コード（必要に応じて）}
  {Suggested fix code (if applicable)}
  
  {参考情報・理由（任意）}
  {Reference information/reasoning (optional)}
  ```

#### 3-3-3. 統合サマリーレビュー作成
インラインコメントでカバーしない全体的な所見をまとめた包括的レビューコメントを生成。

---

## フェーズ 4: GitHubレビュー投稿

### 4-1. 投稿方式の決定と準備
- **具体的指摘の分類**:
  - インラインコメント対象: ファイル・行番号が特定できる問題
  - 全体コメント対象: 全般的な改善点、アーキテクチャレベルの問題
- **投稿戦略**:
  - **Blocking/Should Fix あり** → インラインコメント + `--request-changes` レビュー
  - **Nits のみ** → インラインコメント + 通常コメント
  - **全体的指摘のみ** → 統合レビューコメントのみ

### 4-2. インラインコメントの作成
各エージェントのレビュー結果から、以下の形式でインラインコメント用JSONを生成：

```json
[
  {
    "path": "content/posts/serena.md",
    "line": 84,
    "body": "🚨 **Blocking**: 設定値とコメントが矛盾しています\n🚨 **Blocking**: Configuration value contradicts the comment\n\n```yaml\n# 読み取り専用モードを有効化\n# Enable read-only mode\nread_only: true  # falseではなくtrueにすべき / should be true, not false\n```",
    "start_line": 83,
    "start_side": "RIGHT"
  },
  {
    "path": "content/posts/serena.md", 
    "line": 32,
    "body": "🚨 **Security**: スクリプトを直接実行するのは危険です\n🚨 **Security**: Directly executing scripts is dangerous\n\n```bash\n# より安全な方法 / Safer approach\ncurl -LsSf https://astral.sh/uv/install.sh -o install_uv.sh\ncat install_uv.sh  # 内容確認 / Verify content\nsh install_uv.sh\n```"
  }
]
```

### 4-3. 投稿実行（優先順位順）

#### 4-3-1. インラインコメント投稿
```bash
# 具体的指摘をインラインコメントとして投稿
gh pr review "$ARGUMENTS" $REPO_FLAG --comment --body-file /tmp/inline_comments_body.md

# または、GitHub API を直接使用してインラインコメント作成
for comment in $(cat /tmp/inline_comments.json | jq -r '.[] | @base64'); do
    _jq() {
        echo "${comment}" | base64 --decode | jq -r "${1}"
    }
    gh api repos/$(gh repo view --json nameWithOwner --jq .nameWithOwner)/pulls/"$ARGUMENTS"/comments \
        --method POST \
        --field body="$(_jq '.body')" \
        --field path="$(_jq '.path')" \
        --field line="$(_jq '.line')" \
        --field side="RIGHT"
done
```

#### 4-3-2. 統合レビューコメント投稿
```bash
# 全体的な統合レビューを投稿
if [[ "$HAS_BLOCKING_ISSUES" == "true" ]]; then
    # Blocking issues がある場合は Change Request
    gh pr review "$ARGUMENTS" $REPO_FLAG --request-changes --body-file /tmp/integrated_review.md
else
    # そうでなければ通常コメント
    gh pr comment "$ARGUMENTS" $REPO_FLAG --body-file /tmp/integrated_review.md
fi
```

#### 4-3-3. レビューステータス設定
```bash
# 最終的なレビュー状態を設定
if [[ "$HAS_BLOCKING_ISSUES" == "true" ]]; then
    echo "✋ Changes Requested - インラインコメントで具体的修正点を確認してください"
    echo "✋ Changes Requested - Please check specific fixes in inline comments"
else
    echo "✅ Review Complete - 推奨改善点をインラインコメントで確認してください" 
    echo "✅ Review Complete - Please check recommended improvements in inline comments"
fi
```

### 4-4. 投稿確認
```bash
# インラインコメント数の確認
INLINE_COUNT=$(gh api repos/$(gh repo view --json nameWithOwner --jq .nameWithOwner)/pulls/"$ARGUMENTS"/comments --jq 'length')

# 全体コメント数の確認  
TOTAL_COMMENTS=$(gh pr view "$ARGUMENTS" $REPO_FLAG --json comments --jq '.comments | length')

echo "📊 投稿完了: インライン ${INLINE_COUNT}件, 全体コメント ${TOTAL_COMMENTS}件"
echo "📊 Review Posted: ${INLINE_COUNT} inline comments, ${TOTAL_COMMENTS} general comments"
```

---

## 統合レビューフォーマット

```markdown
## 🤖 AIチーム統合レビュー / Integrated AI Team Review

### 📊 レビュー概要 / Review Summary

この PR は [機能名] の実装です。3つの AI エージェント（Claude、Codex、Gemini）による並列レビューを統合した結果をお報告します。

This PR implements [feature name]. Here are the integrated results from parallel reviews by three AI agents (Claude, Codex, Gemini).

### ✅ 合意された良い点 / Consensus: Good Points

[3つのエージェントが共通して評価した良い点]

### ⚠️ 合意された改善点 / Consensus: Issues to Address

#### 🚨 Blocking（必須修正）
[3エージェント合意の重要問題]

#### 💡 Should Fix（推奨修正）
[3エージェント合意の改善提案]

### 🔍 エージェント個別所見 / Individual Agent Insights

#### Claude Code の所見:
[Claude固有の指摘事項]

#### Codex (GPT-5) の所見:
[Codex固有の指摘事項]

#### Gemini の所見:
[Gemini固有の指摘事項]

### 🤝 最終推奨事項 / Final Recommendations

[統合判断に基づく最終的な推奨アクション]

---
*このレビューは Claude Code の AI チーム並列レビュー機能により生成されました*
*Generated by Claude Code's AI Team Parallel Review feature*
```

---

## エラーハンドリング

- **MCP接続失敗**: 接続できないMCPがあっても、利用可能なエージェントで継続
- **GitHub API エラー**: レート制限時はリトライ、権限エラーは通常コメントモードに切替
- **エージェント実行失敗**: 失敗したエージェントを除外して統合処理を継続

---

## 使用例

```bash
# 現在リポジトリのPRを番号指定
claude ai-team-pr-review 123

# PR URL指定
claude ai-team-pr-review https://github.com/owner/repo/pull/123

# 特定リポジトリを明示
claude ai-team-pr-review 123 --repo owner/repo
```

---

## 期待する品質基準

- **事実と推測の区別**: 差分データに基づく事実と推論を明確に分離
- **具体性**: 抽象的でなく、具体的で検証可能な指摘（ファイル名:行番号付き）
- **優先度の妥当性**: ビジネス影響度に応じた適切な分類
- **インラインコメント活用**: 具体的問題はインライン、全体的問題は統合コメント
- **修正提案の具体性**: 単なる指摘ではなく、実装可能な修正案を提示
- **投稿完了まで**: 分析から GitHub への詳細投稿・確認まで実行

## 投稿結果の期待値

✅ **成功例 / Success Example**:
- インラインコメント / Inline comments: 5-15件（具体的な問題指摘 / specific issue reports）
- 統合レビュー / Integrated review: 1件（全体サマリー / overall summary）
- レビューステータス / Review status: Change Requested または Approved
- 各指摘に修正提案コードを含む / Each comment includes suggested fix code
- **すべてのコメントで日英併記を実施 / All comments must include both Japanese and English**