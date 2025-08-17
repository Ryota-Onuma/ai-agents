---

description: "並列AIチームでPRレビュー: Claude、Codex、Geminiエージェントを同時実行し統合レビューをGitHubに投稿"
argument-hint: "\<pr\_url\_or\_number> \[--repo \<owner/repo>]"
allowed-tools: Bash(gh pr view:*), Bash(gh pr diff:*), Bash(gh pr checks:*), Bash(gh pr review:*), Bash(gh pr comment:*), Task(*), Python(\~/scripts/post-review-command.py)
----------------------------------------------------------------------------------------------------------------------------------------------------------------------------

# AI チーム並列 PR レビューコマンド（修正版）

3 つの AI エージェント（Claude、Codex MCP、Gemini MCP）を並列で実行し、統合されたレビューを GitHub に投稿します。

> **変更点（重要）**: 公式 CLI に `claude mcp call` サブコマンドは存在しないため、**Serena を直接 CLI から呼ぶ行を全廃**しました。Serena 活用は**任意**で、後述のフォールバック（git/grep など）だけでも動作します。

## 実行フロー

1. **前提条件チェック** (並列実行)
2. **PR 情報取得**
   2.5 **関連コードの追加コンテキスト取得（Serena/フォールバック）**
3. **3 エージェント並列レビュー** (Claude、Codex、Gemini)
4. **レビュー統合・優先度付け**
5. **GitHub に投稿**

---

## 入力

- **PR**: `$ARGUMENTS`（PR 番号、URL、またはフルパス）
- **リポジトリ**: `--repo <owner/repo>`（省略時は現在のディレクトリから自動検出）

---

## フェーズ 0: 前提条件チェック（並列実行）

### 0-1. GitHub CLI 認証確認

```bash
gh auth status
```

### 0-2. MCP サーバー接続確認

```bash
# 並列チェック
claude mcp get codex &
claude mcp get gemini &
# Serena MCP は任意（存在すれば活用。ただし本テンプレートでは直接呼び出しは行わない）
claude mcp get serena &
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

## フェーズ 1: PR コンテキスト取得

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

## フェーズ 1.5: 関連コードの追加コンテキスト取得（フェーズ 1 結果を基に）

目的: PR の Diff に現れないが影響しうる箇所を早期に洗い出し、フェーズ 2 の各エージェントに追加文脈として提供する（より本質的なレビューのため）。

### 1.5-1. フェーズ 1 成果の集約（再取得でも可）

```bash
# 作業ディレクトリ
WORK=/tmp/related_ctx
rm -rf "$WORK" && mkdir -p "$WORK"/snippets "$WORK"/logs

# 主要アーティファクトを保存（フェーズ1で取得済みでも、ここで保存しておく）
gh pr view "$ARGUMENTS" $REPO_FLAG \
  --json number,title,author,baseRefName,headRefName,isDraft,mergeable,additions,deletions,changedFiles,url,createdAt,updatedAt,labels,body \
  > "$WORK/pr_meta.json"
gh pr diff "$ARGUMENTS" $REPO_FLAG --name-only --color=never > "$WORK/changed_files.txt"
gh pr diff "$ARGUMENTS" $REPO_FLAG --color=never > "$WORK/diff.patch"
gh pr checks "$ARGUMENTS" $REPO_FLAG > "$WORK/checks.txt"
```

### 1.5-2. Serena MCP を活用した高度な関連探索

Serena MCP の強力なツール群を使用して、PR の変更に関連するコードの深いコンテキストを収集します。

#### 1.5-2-1. Serena ツールによる関連ファイル探索

```bash
# Serena が利用可能な場合の高度なコンテキスト収集
if claude mcp get serena >/dev/null 2>&1; then
  echo "[info] Serena MCP detected. Using advanced context collection tools." | tee "$WORK/logs/serena.txt"

  # 変更ファイルのディレクトリ構造を分析
  while IFS= read -r f; do
    [ -z "$f" ] && continue
    dir=$(dirname "$f")

    # ディレクトリ内のファイル一覧を取得
    echo "=== Directory: $dir ===" >> "$WORK/logs/serena_analysis.txt"
    claude mcp call serena list_dir --path "$dir" >> "$WORK/logs/serena_analysis.txt" 2>&1 || true

    # ファイルのシンボル概要を取得
    if [ -f "$f" ]; then
      echo "=== Symbols in: $f ===" >> "$WORK/logs/serena_analysis.txt"
      claude mcp call serena get_symbols_overview --path "$f" >> "$WORK/logs/serena_analysis.txt" 2>&1 || true
    fi
  done < "$WORK/changed_files.txt"

  # 関連ファイルの検索と分析
  > "$WORK/serena_related_files.txt"
  > "$WORK/serena_symbols.json"

  while IFS= read -r f; do
    [ -z "$f" ] && continue
    base=$(basename "$f")
    stem="${base%.*}"

    # ファイル名ベースでの関連ファイル検索
    echo "=== Finding files related to: $base ===" >> "$WORK/logs/serena_analysis.txt"
    claude mcp call serena find_file --query "$stem" >> "$WORK/logs/serena_analysis.txt" 2>&1 || true

    # シンボルベースでの関連性探索
    if [ -f "$f" ]; then
      # ファイル内の主要シンボルを抽出
      claude mcp call serena get_symbols_overview --path "$f" | jq -r '.symbols[]?.name' 2>/dev/null | while read -r symbol; do
        [ -z "$symbol" ] && continue

        # シンボルを参照している他のファイルを検索
        echo "=== Finding references to symbol: $symbol ===" >> "$WORK/logs/serena_analysis.txt"
        claude mcp call serena find_referencing_symbols --symbol "$symbol" >> "$WORK/logs/serena_analysis.txt" 2>&1 || true

        # シンボルの定義箇所を検索
        echo "=== Finding symbol definition: $symbol ===" >> "$WORK/logs/serena_analysis.txt"
        claude mcp call serena find_symbol --query "$symbol" >> "$WORK/logs/serena_analysis.txt" 2>&1 || true
      done
    fi
  done < "$WORK/changed_files.txt"

  # Serena の分析結果を構造化
  {
    echo "{"
    echo "  \"serena_analysis\": {"
    echo "    \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\","
    echo "    \"changed_files\": ["
    while IFS= read -r f; do
      [ -z "$f" ] && continue
      echo "      \"$f\","
    done < "$WORK/changed_files.txt" | sed '$ s/,$//'
    echo "    ],"
    echo "    \"related_files\": [],"
    echo "    \"symbols\": [],"
    echo "    \"references\": []"
    echo "  }"
    echo "}"
  } > "$WORK/serena_related.json"

  # 関連ファイルのスニペット収集
  mkdir -p "$WORK/serena_snippets"
  while IFS= read -r f; do
    [ -z "$f" ] || [ ! -f "$f" ] && continue

    # ファイルの内容を読み取り（先頭200行）
    echo "=== Reading file: $f ===" >> "$WORK/logs/serena_analysis.txt"
    claude mcp call serena read_file --path "$f" --start_line 1 --end_line 200 >> "$WORK/logs/serena_analysis.txt" 2>&1 || true

    # スニペットとして保存
    mkdir -p "$WORK/serena_snippets/$(dirname "$f")"
    claude mcp call serena read_file --path "$f" --start_line 1 --end_line 200 > "$WORK/serena_snippets/$f" 2>/dev/null || true
  done < "$WORK/changed_files.txt"

  echo "[info] Serena analysis completed. Results saved to $WORK/serena_related.json" | tee -a "$WORK/logs/serena.txt"

else
  echo "[info] Serena MCP not available. Using fallback methods." | tee "$WORK/logs/serena.txt"
fi
```

### 1.5-3. Serena 不在時（または 1.5-2 をスキップ）のヒューリスティック（フォールバック）

```bash
# 変更ファイルから関連候補を収集（同一ディレクトリ、同名、テスト、参照元など）
cp "$WORK/changed_files.txt" "$WORK/changed_files.orig.txt"
> "$WORK/siblings.txt"; > "$WORK/test_candidates.txt"; > "$WORK/importers.txt"; > "$WORK/history.txt"

while IFS= read -r f; do
  [ -z "$f" ] && continue
  dir=$(dirname "$f"); base=$(basename "$f"); stem="${base%.*}"

  # 同一ディレクトリの近傍ファイル
  git ls-files "$dir" >> "$WORK/siblings.txt" || true

  # テスト/Spec/Fixture 候補（一般的な命名）
  git ls-files | grep -E "(/tests?/|^tests?/|_test\.|\.test\.|\.spec\.|/spec/)" | grep -i "$stem" >> "$WORK/test_candidates.txt" || true

  # 参照元（importや文字列参照を緩く検索）
  name_noext=$(echo "$base" | sed 's/\.[^.]*$//')
  git grep -l -n -I -- "$name_noext" -- ":!$f" >> "$WORK/importers.txt" || true

  # 最近の変更履歴
  git log --oneline -n 20 -- "$f" >> "$WORK/history.txt" 2>/dev/null || true

done < "$WORK/changed_files.txt"

# 統合と重複排除（リポジトリ内に限る）
cat "$WORK/siblings.txt" "$WORK/test_candidates.txt" "$WORK/importers.txt" | sort -u > "$WORK/related_candidates.txt"
grep -vxF -f "$WORK/changed_files.txt" "$WORK/related_candidates.txt" > "$WORK/related_files.txt" || cp "$WORK/related_candidates.txt" "$WORK/related_files.txt"

# 代表的なスニペットを収集（先頭200行）
while IFS= read -r rf; do
  [ -f "$rf" ] || continue
  mkdir -p "$WORK/snippets/$(dirname "$rf")"
  sed -n '1,200p' "$rf" > "$WORK/snippets/$rf" 2>/dev/null || true

done < "$WORK/related_files.txt"
```

### 1.5-4. 出力物（フェーズ 2 へ受け渡し）

- 追加コンテキスト（Serena 利用時とフォールバックの統合）:

  - **Serena 分析結果**:

    - `"$WORK/serena_related.json"`（Serena による詳細分析結果）
    - `"$WORK/serena_snippets/"`（Serena が読み取ったファイルスニペット）
    - `"$WORK/logs/serena_analysis.txt"`（Serena ツールの実行ログ）

  - **フォールバック結果**:
    - `"$WORK/related_files.txt"`（git ベースの関連候補ファイル一覧）
    - `"$WORK/snippets/"`（関連候補の先頭 200 行スニペット）
    - `"$WORK/history.txt"`（変更ファイルの近傍履歴）

```bash
# 両方ある場合は統合（重複排除）
if [ -f "$WORK/serena_related.json" ]; then
  jq -r '.related_files[]?' "$WORK/serena_related.json" | sort -u > "$WORK/related_files.serena.txt" || true
fi

if [ -f "$WORK/related_files.serena.txt" ] && [ -f "$WORK/related_files.txt" ]; then
  sort -u "$WORK/related_files.serena.txt" "$WORK/related_files.txt" > "$WORK/related_files_all.txt"
elif [ -f "$WORK/related_files.serena.txt" ]; then
  cp "$WORK/related_files.serena.txt" "$WORK/related_files_all.txt"
elif [ -f "$WORK/related_files.txt" ]; then
  cp "$WORK/related_files.txt" "$WORK/related_files_all.txt"
fi

# 統合リストがあれば不足スニペットを補完
if [ -f "$WORK/related_files_all.txt" ]; then
  while IFS= read -r rf; do
    [ -f "$rf" ] || continue
    out="$WORK/snippets/$rf"
    if [ ! -f "$out" ]; then
      mkdir -p "$(dirname "$out")" && sed -n '1,200p' "$rf" > "$out" 2>/dev/null || true
    fi
  done < "$WORK/related_files_all.txt"
fi
```

---

## フェーズ 2: AI エージェント並列レビュー

### Task tool を使用して 3 エージェントを並列実行

**必ず**以下 3 つのエージェントを使用してください。

1. **claude-reviewer** エージェント

   - Claude Code による直接レビュー
   - 基本的な品質チェック（正当性、セキュリティ、パフォーマンス、可読性）

2. **codex-reviewer** エージェント

   - Codex MCP を使用したレビュー
   - GPT-5 による高度なコード解析

3. **gemini-reviewer** エージェント

   - Gemini MCP を使用したレビュー
   - Gemini による多角的分析

### 共通入力データ

各エージェントに以下の統一コンテキストを提供：

- PR メタデータ（JSON）
- 変更ファイル一覧
- 差分内容（unified diff）
- CI チェック結果
- 追加コンテキスト（フェーズ 1.5 の成果）：

  - **Serena 高度分析結果**:

    - `serena_related.json`（シンボル分析、参照関係、ファイル関連性）
    - `serena_snippets/`（関連ファイルの詳細スニペット）
    - `logs/serena_analysis.txt`（Serena ツール実行ログ）

  - **フォールバック結果**:

    - `related_files.txt`（git ベースの関連候補）
    - `snippets/`（関連候補の先頭 200 行）
    - `history.txt`（変更履歴）

  - **統合結果**: `related_files_all.txt`（Serena + フォールバックの統合）

**注**: これらの入力の取得は本コマンド側で完了させ、各エージェントは個別に再取得しないこと（DRY の徹底）。Serena の分析結果により、より深いコード理解と関連性の把握が可能になります。

### エージェント共通指示事項

各エージェントには以下の形式でレビューを依頼：

**重要**: 🚨 Blocking（必須修正）と 💡 Should Fix（推奨修正）の指摘は、**例外を除き必ずインラインコメントで投稿**します。具体的な問題を発見した場合は、必ず **ファイル名:行番号** を明記してください。

**レビューフォーマット要求**（日英併記必須）：

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

**インラインコメント投稿ルール**:

- 🚨 Blocking（必須修正）: **必ず**インラインコメントで投稿
- 💡 Should Fix（推奨修正）: **必ず**インラインコメントで投稿
- 🔧 Nits（任意改善）: インラインコメントで投稿（推奨）
- 全体的所見: 統合レビューコメントでのみ投稿

---

## フェーズ 3: レビュー結果統合

### 3-1. 各エージェントの結果を収集

- Claude reviewer の所見
- Codex reviewer の所見
- Gemini reviewer の所見

### 3-2. 統合分析実行

- **合意点の抽出**: 3 つのエージェントが共通して指摘する問題
- **相違点の分析**: エージェント間で意見が分かれる箇所
- **優先度付け**:

  - **Blocking（必須修正）**: セキュリティ、重大バグ、仕様逸脱
  - **Should Fix（推奨修正）**: 品質、性能、保守性の改善
  - **Nits（任意改善）**: スタイル、微細最適化
  - **Open Questions（確認事項）**: 追加情報が必要な項目

**注意事項**

- 本質的なレビューになっているか、次ステップに行く前に、think harder で見直すこと。

### 3-3. 統合レビューコメント作成

#### 3-3-1. インラインコメント対象の特定

各エージェントのレビューから、具体的なファイル・行番号が特定できる指摘を抽出：

```bash
# 各エージェントレビューから行番号付き指摘を抽出（例）
grep -n "Line [0-9]*:" /tmp/claude_review.md > /tmp/claude_inline_issues.txt || true
grep -n "行 [0-9]*:" /tmp/codex_review.md > /tmp/codex_inline_issues.txt || true
grep -n "content/posts/.*.md:[0-9]*" /tmp/gemini_review.md > /tmp/gemini_inline_issues.txt || true
```

#### 3-3-2. インラインコメント JSON 生成

（略：元テンプレートの方針に準拠）

#### 3-3-3. 統合サマリーレビュー作成

---

## フェーズ 4: GitHub レビュー投稿

### 4-1. 投稿方式の決定と準備

- **具体的指摘の分類**:

  - **インラインコメント対象（必須）**: 🚨 Blocking（必須修正）、💡 Should Fix（推奨修正）の具体的指摘
  - **インラインコメント対象（推奨）**: 🔧 Nits（任意改善）の具体的指摘
  - **全体コメント対象**: 全般的な改善点、アーキテクチャレベルの問題、エージェント個別所見

- **投稿戦略**:

  - **🚨 Blocking（必須修正）あり** → **必ず**インラインコメント + `--request-changes` レビュー
  - **💡 Should Fix（推奨修正）のみ** → **必ず**インラインコメント + 通常コメント
  - **🔧 Nits（任意改善）のみ** → インラインコメント + 通常コメント
  - **全体的指摘のみ** → 統合レビューコメントのみ

### 4-2. インラインコメントの作成（例）

````json
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
````

### 4-3. 投稿実行（優先順位順）

```bash
# 具体的指摘をインラインコメントとして投稿（例）
gh pr review "$ARGUMENTS" $REPO_FLAG --comment --body-file /tmp/inline_comments_body.md

# または、GitHub API を直接使用してインラインコメント作成（例）
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

```bash
# 全体的な統合レビューを投稿（例）
if [[ "$HAS_BLOCKING_ISSUES" == "true" ]]; then
    gh pr review "$ARGUMENTS" $REPO_FLAG --request-changes --body-file /tmp/integrated_review.md
else
    gh pr comment "$ARGUMENTS" $REPO_FLAG --body-file /tmp/integrated_review.md
fi
```

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
## 🤖 AI チーム統合レビュー / Integrated AI Team Review

### 📊 レビュー概要 / Review Summary

この PR は [機能名] の実装です。3 つの AI エージェント（Claude、Codex、Gemini）による並列レビューを統合した結果をお報告します。

This PR implements [feature name]. Here are the integrated results from parallel reviews by three AI agents (Claude, Codex, Gemini).

### ✅ 合意された良い点 / Consensus: Good Points

[3 つのエージェントが共通して評価した良い点]

### ⚠️ 合意された改善点 / Consensus: Issues to Address

#### 🚨 Blocking（必須修正）

[3 エージェント合意の重要問題]

#### 💡 Should Fix（推奨修正）

[3 エージェント合意の改善提案]

### 🔍 エージェント個別所見 / Individual Agent Insights

#### Claude Code の所見:

[Claude 固有の指摘事項]

#### Codex (GPT-5) の所見:

[Codex 固有の指摘事項]

#### Gemini の所見:

[Gemini 固有の指摘事項]

### 🤝 最終推奨事項 / Final Recommendations

[統合判断に基づく最終的な推奨アクション]

---

_このレビューは Claude Code の AI チーム並列レビュー機能により生成されました_
_Generated by Claude Code's AI Team Parallel Review feature_
```

---

## エラーハンドリング

- **MCP 接続失敗**: 接続できない MCP があっても、利用可能なエージェントで継続
- **GitHub API エラー**: レート制限時はリトライ、権限エラーは通常コメントモードに切替
- **エージェント実行失敗**: 失敗したエージェントを除外して統合処理を継続

---

## 使用例

```bash
# 現在リポジトリのPRを番号指定
claude pr-review-by-ai-team-parallel 123

# PR URL指定
claude pr-review-by-ai-team-parallel https://github.com/owner/repo/pull/123

# 特定リポジトリを明示
claude pr-review-by-ai-team-parallel 123 --repo owner/repo
```

---

## 期待する品質基準

- **事実と推測の区別**: 差分データに基づく事実と推論を明確に分離
- **具体性**: 抽象的でなく、具体的で検証可能な指摘（ファイル名:行番号付き）
- **優先度の妥当性**: ビジネス影響度に応じた適切な分類
- **インラインコメント活用**: 🚨 Blocking（必須修正）と 💡 Should Fix（推奨修正）は**必ず**インラインコメントで投稿、全体的問題は統合コメント
- **修正提案の具体性**: 単なる指摘ではなく、実装可能な修正案を提示
- **投稿完了まで**: 分析から GitHub への詳細投稿・確認まで実行

## 投稿結果の期待値

✅ **成功例 / Success Example**:

- インラインコメント / Inline comments: 5-15 件（🚨 Blocking と 💡 Should Fix の指摘は**必ず**インラインコメント / 🚨 Blocking and 💡 Should Fix issues **must** be inline comments）
- 統合レビュー / Integrated review: 1 件（全体サマリー / overall summary）
- レビューステータス / Review status: Change Requested または Approved
- 各指摘に修正提案コードを含む / Each comment includes suggested fix code
- **すべてのコメントで日英併記を実施 / All comments must include both Japanese and English**
- **🚨 Blocking（必須修正）と 💡 Should Fix（推奨修正）の指摘は例外なくインラインコメントで投稿 / 🚨 Blocking and 💡 Should Fix issues must be posted as inline comments without exception**
