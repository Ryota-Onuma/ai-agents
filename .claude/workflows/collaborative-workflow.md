# 協調ワークフロー設定

エージェント間の協調作業を定義する設定ファイル。

## フェーズ別協調パターン

### 1. 仕様・設計フェーズ
**参加エージェント**: planner, spec-writer, architect, db-migration

**協調順序**:
1. **planner**: GitHub IssueからAcceptance CriteriaとWBS作成
2. **spec-writer**: PRDとTech Spec作成（ユーザーストーリー、非機能要件、成功指標）
3. **architect**: アーキテクチャ、API、データフロー、トレードオフ定義
4. **db-migration**: スキーマ変更設計（前方/後方互換、オンライン移行、ロールバック）

**成果物**:
- 要件仕様書（spec-writer）
- 技術設計書（architect）
- 依存関係を考慮したTODOリスト（planner）
- データベース移行計画（db-migration）

### 2. 実装フェーズ
**参加エージェント**: coder-backend, coder-frontend, test-engineer

**協調パターン**:
- **coder-backend**: SOLID原則厳守、Kotlin/Go優先、安全・小さなコミット
- **coder-frontend**: SOLID原則厳守、React+TypeScript、アクセシビリティ重視
- **test-engineer**: TDD支援（t-wada方式）、テスト設計・実装

**同期ポイント**:
- APIコントラクト調整（backend ↔ frontend）
- 統合テスト実施（test-engineer主導）
- セキュリティ・パフォーマンス確認

### 3. レビューフェーズ
**参加エージェント**: reviewer, planner, spec-writer, architect, db-migration

**レビュープロセス**:
1. **reviewer**: 差分レビューと静的チェック、コード規約・命名・複雑度確認
2. **planner**: 要件充足性確認、WBSとの整合性チェック
3. **spec-writer**: 仕様適合性確認、非機能要件達成度評価
4. **architect**: アーキテクチャ整合性確認、設計原則準拠チェック
5. **db-migration**: スキーマ変更の安全性確認、移行計画との整合性

**承認条件**: **全員のOK**が揃った場合のみPR作成許可

## エージェント間通信プロトコル

### 情報共有形式
```yaml
phase: "design" | "implementation" | "review"
artifacts:
  - type: "requirement" | "design" | "code" | "test" | "migration"
    content: "成果物の内容"
    author: "作成エージェント名"
    dependencies: ["依存する他の成果物"]
status: "pending" | "in_progress" | "completed" | "blocked"
blockers: ["阻害要因のリスト"]
```

### 協調ルール

#### プランニング段階
- GitHub URLとbase branchを必須入力として受け取る
- base branch未指定時はmain branchをデフォルト使用
- 新規ブランチをbase branchから作成

#### 実装段階
- 各エージェントは専門領域内でのみ作業
- 他エージェントの成果物に直接変更を加えない
- 問題発見時は該当エージェントに報告・相談

#### レビュー段階
- 承認条件: 全参加エージェントの明示的なOK
- 1つでもNGがあればPR作成を停止
- 修正後は再度全員レビューを実施

## MCPツール活用

### Serena MCP統合
- https://github.com/oraios/serena をMCPとして活用
- 効率的な作業進行をサポート
- エージェント間のタスク調整に使用

### ツール使用優先度
1. MCP提供ツール（mcp__で始まるもの）を優先
2. 標準ツールは補完的に使用
3. カスタムツールは必要に応じて開発

## エラーハンドリング

### ブロッカー対応
- 技術的制約: architect、coder-*が協調して解決策検討
- 要件不明確: spec-writer、plannerが仕様明確化
- 依存関係問題: planner主導で作業順序再調整

### エスカレーション
- エージェント単体で解決困難な場合、関連エージェント全体で協議
- 最終的にはユーザーに判断を仰ぐ

## 品質保証

### TDD実践
- t-wada推奨手法を全エージェントが遵守
- Red → Green → Refactorサイクル
- E2Eテストは不要（ユニット・統合テストに注力）

### SOLID原則
- 全実装エージェントが厳守
- レビュー時に原則遵守を確認
- コンテキスト・可読性も粒度決定要因として考慮