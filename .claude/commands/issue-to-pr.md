---
allowed-tools:
  # --- Git 基本操作（非破壊系） ---
  - Bash(git clone:*)
  - Bash(git fetch:*)
  - Bash(git checkout:*)
  - Bash(git switch:*)
  - Bash(git branch:*)
  - Bash(git add:*)
  - Bash(git commit:*)
  - Bash(git push:*)
  - Bash(git status:*)
  - Bash(git diff:*)
  - Bash(git log:*)

  # --- GitHub CLI ---
  - Bash(gh issue view:*)
  - Bash(gh pr create:*)
  - Bash(gh pr view:*)
  - Bash(gh pr edit:*)

  # --- pnpm ---
  - Bash(pnpm install)
  - Bash(pnpm run build)
  - Bash(pnpm run test)
  - Bash(pnpm run lint)
  - Bash(pnpm run format)
  - Bash(pnpm run typecheck)

  # --- Gradle ---
  - Bash(./gradlew build:*)
  - Bash(./gradlew test:*)
  - Bash(./gradlew tasks:*)
  - Bash(./gradlew ktlint*)
  - Bash(./gradlew detekt*)

  # --- TypeScript / Lint / Format ---
  - Bash(tsc:*)
  - Bash(eslint:*)
  - Bash(prettier:*)
description: "GitHub IssueからPR作成までの全自動ワークフロー（pnpm + Gradle対応）"
---

# Issue to PR Command

GitHub IssueからPR作成まで全自動化されたワークフローコマンド。

## 使用方法

```bash
# 基本使用（main branchベース）
issue-to-pr <github_issue_url>

# 特定ブランチベース
issue-to-pr <github_issue_url> --base <branch_name>

## 引数

- `github_issue_url` (必須): GitHub Issue URL
  - 形式: `https://github.com/owner/repo/issues/123`
- `--base` (オプション): ベースブランチ名（デフォルト: main）

## 実行フロー

### Phase 1: 分析・計画 (Sequential)
1. **planner**: Issue分析、Acceptance Criteria作成、WBS作成
2. **spec-writer**: PRD・Tech Spec作成、非機能要件定義
3. **architect**: アーキテクチャ設計、API設計、影響範囲評価
4. **db-migration**: スキーマ変更設計、移行戦略策定

### Phase 2: 実装 (Parallel + Sequential)
1. **pr-bot**: 新規ブランチ作成（`feature/issue-{number}-{description}`）
2. **並列実装**:
   - **backend-expert**: SOLID原則、Kotlin/Go、TDD実践
   - **frontend-expert**: SOLID原則、React+TypeScript、アクセシビリティ
   - **test-engineer**: t-wada方式TDD支援、テスト設計・実装

### Phase 3: レビュー (Approval Gate)
**全員承認必須** - 1つでもNGがあればPR作成停止
1. **reviewer**: コード品質、規約、命名、複雑度
2. **planner**: 要件充足性、WBS整合性
3. **spec-writer**: 仕様適合性、非機能要件達成
4. **architect**: アーキテクチャ整合性、設計原則準拠
5. **db-migration**: スキーマ変更安全性、移行計画整合性

### Phase 4: 統合
1. **pr-bot**: PR作成、Issue紐付け、CI/CD実行

## エージェント協調ルール

### 情報共有
- 各エージェントは専門成果物を作成・共有
- 依存関係を明示し、ブロッカーを報告
- 他エージェントの成果物への直接変更禁止

### 品質基準
- **SOLID原則**: 全実装で厳守
- **TDD**: t-wada方式（Red→Green→Refactor）
- **セキュリティ**: 機密情報ログ出力禁止、脆弱性対策
- **テストカバレッジ**: >80%

### MCPツール活用
- Serena MCP (https://github.com/oraios/serena) 優先使用
- 効率的タスク調整、進捗可視化

## エラーハンドリング

### 入力検証
- GitHub URL形式チェック
- Issue存在確認
- ベースブランチ存在確認
- リポジトリアクセス権確認

### 復旧戦略
- フェーズ単位での再実行
- エージェント協議による問題解決
- 必要時のユーザーエスカレーション

## 成果物

- **技術仕様書** (spec-writer)
- **アーキテクチャ設計書** (architect)
- **実装済みコード** (backend-expert, frontend-expert)
- **テストスイート** (test-engineer)
- **データベース移行スクリプト** (db-migration)
- **レビュー報告書** (reviewer)
- **GitHub PR** (pr-bot)

## ワークフロー設定

このコマンドは以下の設定ファイルを参照して動作します：

### 協調ワークフロー
- **設定ファイル**: `.claude/workflows/collaborative-workflow.md`
- **役割**: エージェント間の協調ルール、承認ゲート、品質基準を定義

### エージェント設定
各フェーズで使用されるエージェントの詳細設定：
- `.claude/agents/planner.md`
- `.claude/agents/spec-writer.md`
- `.claude/agents/architect.md`
- `.claude/agents/db-migration.md`
- `.claude/agents/pr-bot.md`
- `.claude/agents/backend-expert.md`
- `.claude/agents/frontend-expert.md`
- `.claude/agents/test-engineer.md`
- `.claude/agents/reviewer.md`

## 実行設定

```yaml
command: issue-to-pr
description: "GitHub IssueからPR作成までの全自動ワークフロー"

workflow_config: ".claude/workflows/collaborative-workflow.md"

phases:
  planning:
    agents: [planner, spec-writer, architect, db-migration]
    execution: sequential

  implementation:
    agents: [pr-bot, backend-expert, frontend-expert, test-engineer]
    execution: mixed  # pr-bot sequential, others parallel

  review:
    agents: [reviewer, planner, spec-writer, architect, db-migration]
    execution: parallel
    approval_gate: all_required

  integration:
    agents: [pr-bot]
    execution: sequential
    conditions: [review_approved]

quality_enforcement:
  workflow_rules: ".claude/workflows/collaborative-workflow.md"
  agent_configs: ".claude/agents/*.md"
```
