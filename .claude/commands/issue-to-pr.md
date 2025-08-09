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
description: "GitHub IssueからPR作成までの全自動ワークフロー"
---

# Issue to PR Command

GitHub Issue から PR 作成まで全自動化されたワークフローコマンド。

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

### Phase 0: ブランチ作成 (Sequential)
1. **pr-bot**: GitHub Issue URL と base branch を受け取り
2. **pr-bot**: 新規ブランチ作成（`issue/{number}-{description}`）を base branch から作成
3. **pr-bot**: 作業環境の初期化（ブランチ切り替え、依存関係整理）

### Phase 1: 分析・計画 (Sequential)

#### プロダクトオーナーチーム
1. **chief-product-owner**: Issue分析、初期要件整理
2. **並列**: product-owner-ux（UX特化仕様）、product-owner-tech（技術特化仕様）
3. **chief-product-owner**: 統合・最終化 → requirements.md

#### アーキテクトチーム
1. **architect-impact**: 既存システム影響調査
2. **並列**: architect-product（プロダクト観点設計）、architect-tech（技術観点設計）
3. **chief-architect + architect-impact**: 統合・最終化 → design.md
4. **db-migration**: スキーマ変更設計、移行戦略策定

### Phase 2: 実装 (Parallel)
1. **並列実装・テスト一体チーム** (実装とテストの一体化):
   - **backend-expert**: 
     - **Capabilities**: backend-development, backend-architecture, technical-architecture, backend-testing, tdd-methodology, solid-principles
     - **実装範囲**: バックエンド実装 + ユニット/統合/API/DB/セキュリティテスト
     - **品質基準**: TDD (t-wada 方式)、SOLID 原則、カバレッジ ≥ 85%
   - **frontend-expert**: 
     - **Capabilities**: frontend-development, frontend-architecture, product-architecture, frontend-testing, tdd-methodology, solid-principles
     - **実装範囲**: フロントエンド実装 + ユニット/統合/UI/UX/アクセシビリティテスト
     - **品質基準**: TDD (t-wada 方式)、SOLID 原則、React+TypeScript、カバレッジ ≥ 80%

### Phase 3: レビュー (Approval Gate)
**全員承認必須** - 1つでもNGがあればPR作成停止
1. **reviewer**: コード品質、規約、命名、複雑度
2. **chief-product-owner**: 要件充足性、requirements.md整合性
3. **chief-architect**: アーキテクチャ整合性、design.md準拠
4. **db-migration**: スキーマ変更安全性、移行計画整合性

### Phase 4: 統合
1. **pr-bot**: PR作成、Issue紐付け、CI/CD実行

## エージェント協調ルール

### 情報共有・非同期通信
- **サブエージェント間通信プロトコル** (`.claude/desk/memory/PROTOCOL.md`) 活用
- **CASストレージ**: 大きな成果物を `cas/sha256/<hash>` で共有
- **NDJSONメッセージング**: `queues/*.inbox.ndjson`/`outbox/*.outbox.ndjson`
- **barriersファイル**: フェーズ同期ポイント管理
- 依存関係を明示し、ブロッカーを報告
- 他エージェントの成果物への直接変更禁止

### 品質基準・Capabilities活用
- **SOLID原則** ([solid-principles.md](.claude/capabilities/solid-principles.md)): 全実装で厳守
- **TDD** ([tdd-methodology.md](.claude/capabilities/tdd-methodology.md)): t-wada方式（Red→Green→Refactor）
- **コード品質基準** ([code-quality-standards.md](.claude/capabilities/code-quality-standards.md)): カバレッジ>80%、複雑度<10
- **協調開発** ([collaborative-development.md](.claude/capabilities/collaborative-development.md)): チーム連携・継続改善
- **アーキテクチャ** (capabilities/*-architecture.md): 各エージェントが専門領域で高度な設計理解力
- **セキュリティ**: 機密情報ログ出力禁止、脆弱性対策

### MCPツール・非同期通信活用
- **Serena MCP** (https://github.com/oraios/serena) 優先使用
- **サブエージェント間通信プロトコル** による効率的タスク調整
- **CAS・barriers・queues** による堅牢な非同期処理
- 進捗可視化・フェーズ管理の自動化

## エラーハンドリング

### 入力検証・前提条件
- **GitHub URL形式チェック**: `https://github.com/owner/repo/issues/123`
- **Issue存在確認**: アクセス可能性とステータス確認
- **ベースブランチ存在確認**: 指定ブランチの存在と最新性
- **リポジトリアクセス権確認**: push権限、PR作成権限
- **サブエージェント間通信環境**: `.claude/desk/memory/` ディレクトリ構造確認

### 復旧戦略・障害対応
- **barriersファイル** でのフェーズ単位再実行
- **receipts** でのack管理と再送防止
- **locks** でのリソース競合回避
- **TTL** でのタイムアウト処理
- エージェント協議による問題解決
- 必要時のユーザーエスカレーション

## 成果物

### プロダクトオーナーチーム
- **要件仕様書** (chief-product-owner): `.claude/desk/outputs/requirements/ISSUE-<number>.requirements.md`
- **UX特化仕様** (product-owner-ux): UX観点からの詳細仕様
- **技術特化仕様** (product-owner-tech): 技術観点からの詳細仕様

### アーキテクトチーム
- **設計書** (chief-architect): `.claude/desk/outputs/design/ISSUE-<number>.design.md`
- **影響調査レポート** (architect-impact): 既存システムへの影響分析
- **プロダクト観点設計** (architect-product): UX/価値创出重視の設計
- **技術観点設計** (architect-tech): 技術品質/運用性重視の設計
- **ADR** (chief-architect): `.claude/desk/outputs/adr/ADR-<date>-<slug>.md`

### 実装チーム (実装・テスト一体化)
- **backend-expert**: バックエンド実装 + ユニット/統合/API/DB/セキュリティテスト
- **frontend-expert**: フロントエンド実装 + ユニット/統合/UI/UX/アクセシビリティテスト

### その他
- **データベース移行スクリプト** (db-migration): `.claude/desk/outputs/migrations/`
- **レビュー報告書** (reviewer): コード品質・規約・命名・複雑度チェック
- **承認ゲート管理** (pr-bot): `.claude/desk/outputs/reviews/APPROVALS-ISSUE-<number>.md`
- **GitHub PR** (pr-bot): Issue紐付け、ラベル/レビュワー設定

## ワークフロー設定

このコマンドは以下の設定ファイルを参照して動作します：

### 協調ワークフロー
- **設定ファイル**: `.claude/workflows/collaborative-workflow.md`
- **役割**: エージェント間の協調ルール、承認ゲート、品質基準を定義

### エージェント設定
各フェーズで使用されるエージェントの詳細設定：
- `.claude/agents/chief-product-owner.md`
- `.claude/agents/product-owner-ux.md`
- `.claude/agents/product-owner-tech.md`
- `.claude/agents/chief-architect.md`
- `.claude/agents/architect-impact.md`
- `.claude/agents/architect-product.md`
- `.claude/agents/architect-tech.md`
- `.claude/agents/db-migration.md`
- `.claude/agents/pr-bot.md`
- `.claude/agents/backend-expert.md`
- `.claude/agents/frontend-expert.md`
- `.claude/agents/reviewer.md`

## 実行設定

```yaml
command: issue-to-pr
description: "GitHub IssueからPR作成までの全自動ワークフロー"

workflow_config: ".claude/workflows/collaborative-workflow.md"

phases:
  branch_creation:
    agents: [pr-bot]
    execution: sequential
    description: "ブランチ作成、作業環境初期化"
    
  planning:
    product_owner_team:
      agents: [chief-product-owner, product-owner-ux, product-owner-tech]
      execution: sequential_with_parallel  # chief leads, others parallel, chief finalizes
      output: ".claude/desk/outputs/requirements/ISSUE-<number>.requirements.md"
      capabilities: [collaborative-development, solid-principles, code-quality-standards]
    
    architect_team:
      agents: [architect-impact, architect-product, architect-tech, chief-architect]
      execution: sequential_with_parallel  # impact first, others parallel, chief finalizes
      output: ".claude/desk/outputs/design/ISSUE-<number>.design.md"
      capabilities: [impact-analysis, product-architecture, technical-architecture, architecture-integration]
      
    db_migration:
      agents: [db-migration]
      execution: sequential
      depends_on: [architect_team]
      output: ".claude/desk/outputs/migrations/"

  implementation:
    branch_creation:
      agents: [pr-bot]
      execution: sequential
    development_teams:
      agents: [backend-expert, frontend-expert]
      execution: parallel  # 実装・テスト一体化エージェントが並列作業
      capabilities: [backend-development, frontend-development, backend-testing, frontend-testing, backend-architecture, frontend-architecture, product-architecture, technical-architecture, solid-principles, tdd-methodology]

  review:
    agents: [reviewer, chief-product-owner, chief-architect, db-migration]
    execution: parallel
    approval_gate: all_required
    approval_file: ".claude/desk/outputs/reviews/APPROVALS-ISSUE-<number>.md"

  integration:
    agents: [pr-bot]
    execution: sequential
    conditions: [review_approved]
    github_integration: true

communication_protocol:
  protocol_file: ".claude/desk/memory/PROTOCOL.md"
  cas_storage: ".claude/desk/memory/cas/"
  message_queues: ".claude/desk/memory/queues/"
  barriers: ".claude/desk/memory/barriers/"
  locks: ".claude/desk/memory/locks/"
  receipts: ".claude/desk/memory/receipts/"

quality_enforcement:
  workflow_rules: ".claude/workflows/collaborative-workflow.md"
  agent_configs: ".claude/agents/*.md"
  capabilities: ".claude/capabilities/*.md"
  output_directory: ".claude/desk/outputs/"
```
