# issue-to-pr-universal Command

GitHub Issue から PR 作成まで全自動化されたワークフローコマンド（汎用 LLM 対応版）

## 使用方法

```bash
# 基本使用（main branchベース）
issue-to-pr-universal <github_issue_url>

# 特定ブランチベース
issue-to-pr-universal <github_issue_url> --base <branch_name>
```

## 引数

- `github_issue_url` (必須): GitHub Issue URL
  - 形式: `https://github.com/owner/repo/issues/123`
- `--base` (オプション): ベースブランチ名（デフォルト: main）

## 実行フロー（シーケンシャル実行）

### Phase 0: ブランチ作成

1. **pr-bot エージェントを呼び出し**:
   ```
   エージェント: pr-bot
   入力: GitHub Issue URL, base branch
   タスク: 新規ブランチ作成（`issue/{number}-{description}`）をbase branchから作成、作業環境の初期化
   出力: ブランチ作成完了報告
   ```

### Phase 1: 分析・計画

#### 1.1 プロダクトオーナーチーム（順次実行）

1. **chief-product-owner エージェントを呼び出し**:

   ```
   エージェント: chief-product-owner
   入力: GitHub Issue URL, Issue内容
   タスク: Issue分析、初期要件整理
   出力: 初期要件分析結果
   ```

2. **product-owner-ux エージェントを呼び出し**:

   ```
   エージェント: product-owner-ux
   入力: 初期要件分析結果
   タスク: UX特化仕様の詳細化
   出力: UX仕様書
   ```

3. **product-owner-tech エージェントを呼び出し**:

   ```
   エージェント: product-owner-tech
   入力: 初期要件分析結果
   タスク: 技術特化仕様の詳細化
   出力: 技術仕様書
   ```

4. **chief-product-owner エージェントを再呼び出し**:
   ```
   エージェント: chief-product-owner
   入力: UX仕様書, 技術仕様書
   タスク: 統合・最終化
   出力: requirements.md
   保存先: ~/.claude/desk/outputs/requirements/ISSUE-<number>.requirements.md
   ```

#### 1.2 アーキテクトチーム（順次実行）

1. **architect-impact エージェントを呼び出し**:

   ```
   エージェント: architect-impact
   入力: requirements.md
   タスク: 既存システム影響調査
   出力: 影響調査レポート
   ```

2. **architect-product エージェントを呼び出し**:

   ```
   エージェント: architect-product
   入力: requirements.md, 影響調査レポート
   タスク: プロダクト観点設計
   出力: プロダクト設計書
   ```

3. **architect-tech エージェントを呼び出し**:

   ```
   エージェント: architect-tech
   入力: requirements.md, 影響調査レポート
   タスク: 技術観点設計、DBマイグレーション戦略
   出力: 技術設計書
   ```

4. **chief-architect エージェントを呼び出し**:
   ```
   エージェント: chief-architect
   入力: 影響調査レポート, プロダクト設計書, 技術設計書
   タスク: 統合・最終化
   出力: design.md, ADR
   保存先:
     - ~/.claude/desk/outputs/design/ISSUE-<number>.design.md
     - ~/.claude/desk/outputs/adr/ADR-<date>-<slug>.md
   ```

### Phase 2: 実装計画

1. **implementation-planner エージェントを呼び出し**:
   ```
   エージェント: implementation-planner
   入力: requirements.md, design.md
   タスク: 依存関係を考慮した実装TODOリスト作成
   出力:
     - 実装計画書
     - 進捗チェックリスト
   保存先:
     - ~/.claude/desk/outputs/implementation/ISSUE-<number>.implementation-plan.md
     - ~/.claude/desk/outputs/implementation/ISSUE-<number>.progress.md
   ```

### Phase 3: 実装（順次実行 + 進捗管理）

1. **implementation-tracker エージェントを呼び出し**:

   ```
   エージェント: implementation-tracker
   入力: 実装計画書, 進捗チェックリスト
   タスク: 進捗管理準備、初期状態設定
   出力: 進捗管理開始通知
   ```

2. **backend-expert エージェントを呼び出し**:

   ```
   エージェント: backend-expert
   入力: 実装計画書, design.md
   タスク: バックエンド実装 + DB マイグレーション + テスト（ユニット/統合/API/DB/セキュリティ）
   品質基準: TDD、SOLID原則、カバレッジ ≥ 85%
   出力: バックエンド実装完了報告
   ```

3. **implementation-tracker エージェントを呼び出し**:

   ```
   エージェント: implementation-tracker
   入力: バックエンド実装完了報告
   タスク: 進捗チェックリスト更新、次タスク判定
   出力: 進捗更新完了
   ```

4. **frontend-expert エージェントを呼び出し**:

   ```
   エージェント: frontend-expert
   入力: 実装計画書, design.md, バックエンド実装結果
   タスク: フロントエンド実装 + テスト（ユニット/統合/UI/UX）
   品質基準: TDD、SOLID原則、React+TypeScript、カバレッジ ≥ 80%
   出力: フロントエンド実装完了報告
   ```

5. **implementation-tracker エージェントを呼び出し**:
   ```
   エージェント: implementation-tracker
   入力: フロントエンド実装完了報告
   タスク: 最終進捗チェックリスト更新
   出力: 実装フェーズ完了報告
   ```

### Phase 4: レビュー（承認ゲート）

**全員承認必須** - 1 つでも NG があれば PR 作成停止

1. **reviewer エージェントを呼び出し**:

   ```
   エージェント: reviewer
   入力: 全実装コード
   タスク: コード品質、規約、命名、複雑度チェック
   出力: レビュー報告書
   ```

2. **chief-product-owner エージェントを呼び出し**:

   ```
   エージェント: chief-product-owner
   入力: 全実装結果, requirements.md
   タスク: 要件充足性、requirements.md整合性チェック
   出力: 要件承認結果
   ```

3. **chief-architect エージェントを呼び出し**:

   ```
   エージェント: chief-architect
   入力: 全実装結果, design.md
   タスク: アーキテクチャ整合性、design.md準拠、DBマイグレーション戦略適合性チェック
   出力: アーキテクチャ承認結果
   ```

4. **承認結果の統合**:
   ```
   入力: レビュー報告書, 要件承認結果, アーキテクチャ承認結果
   判定: 全て承認の場合のみ次フェーズへ進行
   出力: 承認ゲート管理ファイル
   保存先: ~/.claude/desk/outputs/reviews/APPROVALS-ISSUE-<number>.md
   ```

### Phase 5: 統合

1. **pr-bot エージェントを呼び出し**:
   ```
   エージェント: pr-bot
   入力: 承認ゲート管理ファイル
   タスク: PR作成、Issue紐付け、CI/CD実行
   出力: PR URL
   ```

## エージェント設定ファイル

各エージェントは以下の設定ファイルに基づいて動作:

- `~/.claude/agents/chief-product-owner.md`
- `~/.claude/agents/product-owner-ux.md`
- `~/.claude/agents/product-owner-tech.md`
- `~/.claude/agents/chief-architect.md`
- `~/.claude/agents/architect-impact.md`
- `~/.claude/agents/architect-product.md`
- `~/.claude/agents/architect-tech.md`
- `~/.claude/agents/implementation-planner.md`
- `~/.claude/agents/implementation-tracker.md`
- `~/.claude/agents/backend-expert.md`
- `~/.claude/agents/frontend-expert.md`
- `~/.claude/agents/reviewer.md`
- `~/.claude/agents/pr-bot.md`

## 品質基準・Capabilities

- **SOLID 原則**: 全実装で厳守
- **TDD**: t-wada 方式（Red→Green→Refactor）
- **コード品質**: カバレッジ>80%、複雑度<10
- **セキュリティ**: 機密情報ログ出力禁止、脆弱性対策

## エラーハンドリング

### 入力検証

- GitHub URL 形式チェック: `https://github.com/owner/repo/issues/123`
- Issue 存在確認とアクセス可能性
- ベースブランチ存在確認と最新性
- リポジトリアクセス権確認（push 権限、PR 作成権限）

### 復旧戦略

- フェーズ単位での再実行対応
- エージェント間のデータ受け渡し管理
- エラー発生時のユーザーエスカレーション

## 成果物

### プロダクトオーナーチーム

- **要件仕様書**: `~/.claude/desk/outputs/requirements/ISSUE-<number>.requirements.md`
- **UX 特化仕様**: UX 観点からの詳細仕様
- **技術特化仕様**: 技術観点からの詳細仕様

### アーキテクトチーム

- **設計書**: `~/.claude/desk/outputs/design/ISSUE-<number>.design.md`
- **影響調査レポート**: 既存システムへの影響分析
- **プロダクト観点設計**: UX/価値創出重視の設計
- **技術観点設計**: 技術品質/運用性重視の設計
- **ADR**: `~/.claude/desk/outputs/adr/ADR-<date>-<slug>.md`

### 実装チーム

- **バックエンド実装**: バックエンド実装 + ユニット/統合/API/DB/セキュリティテスト
- **フロントエンド実装**: フロントエンド実装 + ユニット/統合/UI/UX テスト

### その他

- **実装計画書**: `~/.claude/desk/outputs/implementation/ISSUE-<number>.implementation-plan.md`
- **進捗チェックリスト**: `~/.claude/desk/outputs/implementation/ISSUE-<number>.progress.md`
- **データベース移行スクリプト**: `~/.claude/desk/outputs/migrations/`
- **レビュー報告書**: コード品質・規約・命名・複雑度チェック
- **承認ゲート管理**: `~/.claude/desk/outputs/reviews/APPROVALS-ISSUE-<number>.md`
- **GitHub PR**: Issue 紐付け、ラベル/レビュワー設定

## 使用ツール

### Git 基本操作

- git clone, fetch, checkout, switch, branch
- git add, commit, push
- git status, diff, log

### GitHub CLI

- gh issue view
- gh pr create, view, edit

### ビルドツール

- pnpm install, build, test, lint, format, typecheck
- ./gradlew build, test, tasks, ktlint, detekt

### TypeScript / Lint / Format

- tsc, eslint, prettier

## 実行例

```bash
# Issue #123 に対するPR作成
issue-to-pr-universal https://github.com/owner/repo/issues/123

# develop ブランチベースでPR作成
issue-to-pr-universal https://github.com/owner/repo/issues/123 --base develop
```

このワークフローは、Claude Code 以外の LLM でも同等のパフォーマンスを発揮できるよう、各フェーズでサブエージェントを明示的に順次呼び出す設計になっています。
