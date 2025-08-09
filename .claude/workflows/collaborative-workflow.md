# 協調ワークフロー設定

エージェント間の協調作業を定義する設定ファイル。

言語ポリシー: 常に日本語で回答

## フェーズ別協調パターン

### 1. 仕様・設計フェーズ

#### プロダクトオーナーチーム

**参加エージェント**: chief-product-owner, product-owner-ux, product-owner-tech

**協調順序** (サブエージェント間通信プロトコル使用):

1. **chief-product-owner**: GitHub Issue 分析、初期要件整理
2. **並列処理**: 
   - **product-owner-ux**: UX/ユーザー体験特化仕様作成
   - **product-owner-tech**: 技術/非機能要件特化仕様作成
3. **chief-product-owner**: 各専門家の成果物を統合・レビュー・最終化

#### アーキテクトチーム

**参加エージェント**: chief-architect, architect-impact, architect-product, architect-tech

**協調順序** (サブエージェント間通信プロトコル使用):

1. **architect-impact**: 既存システム影響調査、リスク評価
2. **並列処理**: 
   - **architect-product**: プロダクト価値重視の設計 (UX/ロードマップ/価値仮説との整合)
   - **architect-tech**: 技術品質重視の設計 (性能/可用性/セキュリティ/観測性/運用)
3. **chief-architect + architect-impact**: 各専門家の成果物を統合・レビュー・最終化

#### データベース移行

**参加エージェント**: db-migration

**実施事項**: アーキテクトチームの成果物を受けてスキーマ変更設計、移行戦略策定

**成果物**:

- **要件仕様書** (chief-product-owner): `.claude/desk/outputs/requirements/ISSUE-<number>.requirements.md`
- **設計書** (chief-architect): `.claude/desk/outputs/design/ISSUE-<number>.design.md`
- **ADR** (chief-architect): `.claude/desk/outputs/adr/ADR-<date>-<slug>.md`
- **データベース移行計画** (db-migration): `.claude/desk/outputs/migrations/`

### 2. 実装フェーズ

**参加エージェント**: pr-bot, backend-expert, frontend-expert

**協調パターン** (実装とテストの一体化):

#### ブランチ作成
- **pr-bot**: 新規ブランチ作成 (`issue/{number}-{description}`)

#### 並列実装・テスト一体チーム
- **backend-expert**: 
  - **Capabilities**: [backend-development](../capabilities/backend-development.md), [backend-architecture](../capabilities/backend-architecture.md), [technical-architecture](../capabilities/technical-architecture.md), [backend-testing](../capabilities/backend-testing.md), [tdd-methodology](../capabilities/tdd-methodology.md), [solid-principles](../capabilities/solid-principles.md)
  - **実施事項**: バックエンド実装 + ユニット/統合/API/データベース/セキュリティテスト
  - **品質基準**: SOLID 原則、TDD (t-wada 方式)、Kotlin/Go 優先、カバレッジ ≥ 85%

- **frontend-expert**: 
  - **Capabilities**: [frontend-development](../capabilities/frontend-development.md), [frontend-architecture](../capabilities/frontend-architecture.md), [product-architecture](../capabilities/product-architecture.md), [frontend-testing](../capabilities/frontend-testing.md), [tdd-methodology](../capabilities/tdd-methodology.md), [solid-principles](../capabilities/solid-principles.md)
  - **実施事項**: フロントエンド実装 + ユニット/統合/UI/UX/アクセシビリティテスト
  - **品質基準**: SOLID 原則、TDD (t-wada 方式)、React+TypeScript、カバレッジ ≥ 80%

**同期ポイント** (サブエージェント間通信プロトコル使用):

- **API コントラクト調整**: backend-expert ↔ frontend-expert (各エージェントがテストも含めて完結性を保証)
- **統合テスト実施**: backend-expert ↔ frontend-expert (エンドツーエンドテストの協調)
- **セキュリティ・パフォーマンス確認**: 各エージェントが自分の領域で完全な品質保証

### 3. レビューフェーズ

**参加エージェント**: reviewer, chief-product-owner, chief-architect, db-migration

**レビュープロセス** (並列実行):

1. **reviewer**: 差分レビューと静的チェック、コード規約・命名・複雑度確認
2. **chief-product-owner**: 要件充足性確認、requirements.md との整合性チェック
3. **chief-architect**: アーキテクチャ整合性確認、design.md 準拠チェック
4. **db-migration**: スキーマ変更の安全性確認、移行計画との整合性

**承認ゲート管理**:
- **承認ファイル**: `.claude/desk/outputs/reviews/APPROVALS-ISSUE-<number>.md`
- **承認条件**: **全員の `approved`** が揃った場合のみ PR 作成許可
- **1つでも NG** があれば PR 作成停止
- **修正後は再度全員レビュー実施**

### 4. 統合フェーズ

**参加エージェント**: pr-bot

**実施事項**:
- **PR 作成**: `gh pr create` で GitHub PR 作成
- **Issue 紐付け**: `Closes #<number>` で自動クローズ
- **ラベル/レビュワー設定**: プロジェクト規約に従って設定
- **CI/CD 実行**: GitHub Actions 等の自動テスト・デプロイ

## エージェント間通信プロトコル

プロトコル（`.claude/desk/memory/PROTOCOL.md`）を使用してサブエージェント間通信を行う。

### 協調ルール

#### プランニング段階

- GitHub URL と base branch を必須入力として受け取る
- base branch 未指定時は main branch をデフォルト使用
- 新規ブランチを base branch から作成

#### 実装段階

- 各エージェントは専門領域内でのみ作業
- 他エージェントの成果物に直接変更を加えない
- 問題発見時は該当エージェントに報告・相談

#### レビュー段階

- 承認条件: 全参加エージェントの明示的な OK
- 1 つでも NG があれば PR 作成を停止
- 修正後は再度全員レビューを実施

## MCP ツール活用

### Serena MCP 統合

- https://github.com/oraios/serena を MCP として活用
- 効率的な作業進行をサポート
- エージェント間のタスク調整に使用

### ツール使用優先度

1. MCP 提供ツール（mcp\_\_で始まるもの）を優先
2. 標準ツールは補完的に使用
3. カスタムツールは必要に応じて開発

## エラーハンドリング

### ブロッカー対応・エラーハンドリング

#### サブエージェント間通信レベルのエラー対応
- **TTL タイムアウト**: メッセージの自動無効化、再送機能
- **locks 競合**: リソースロックの原子的取得・解放
- **barriers 管理**: フェーズ同期ポイントでの進行制御
- **receipts ack**: 処理完了確認と再送防止

#### エージェントレベルのブロッカー対応
- **技術的制約**: chief-architect、backend-expert、frontend-expert が協調して解決策検討
- **実装品質問題**: 各 expert エージェントが実装とテストを一体で管理、TDD で品質保証
- **API コントラクト競合**: backend-expert ↔ frontend-expert 間でコントラクト調整
- **要件不明確**: chief-product-owner が product-owner-ux/tech と連携して仕様明確化
- **設計間の競合**: chief-architect が architect-impact/product/tech と連携して調整
- **データベース移行問題**: db-migration がリスク評価と安全な移行戦略再検討

### エスカレーション・異常時対応

#### エージェントレベルエスカレーション
- **チーム内協議**: chief-* エージェントが統括してチーム内協議を主導
- **チーム間協議**: プロダクトオーナーチーム ↔ アーキテクトチームの連携
- **全エージェント協議**: 特に複雑な技術的制約やビジネス要件での競合

#### ユーザーエスカレーション
- **最終判断**: エージェント協議で解決困難な場合、ユーザーに判断を仰ぐ
- **総合的なコンテキスト提供**: 各エージェントの観点と成果物を統合して情報提供
- **実装優先度の再評価**: ビジネス価値、技術的リスク、リソース制約のバランス

## 品質保証

### TDD 実践

- t-wada 推奨手法を全エージェントが遵守
- Red → Green → Refactor サイクル
- UI の包括的自動化テストは対象外（ユニット・統合テストに注力）

### SOLID 原則

- 全実装エージェントが厳守
- レビュー時に原則遵守を確認
- コンテキスト・可読性も粒度決定要因として考慮
