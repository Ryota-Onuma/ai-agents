# 協調開発能力

チーム開発における効果的な協調作業の実践。サブエージェント間通信プロトコルを活用した現代的な協調ワークフロー。

## 設計フェーズでの協調

### プロダクトオーナーチームとの連携
- chief-product-owner からの requirements.md 受領・理解
- product-owner-ux、product-owner-tech の専門知見を実装に反映

### アーキテクトチームとの連携
- chief-architect からの design.md 受領・理解
- architect-impact、architect-product、architect-tech の設計決定を実装に落とし込み
- API コントラクトやインターフェース定義の合意

## 実装フェーズでの協調

- 小さく安全なコミット単位での開発
- 型安全性とエラーハンドリングの重視
- 他チームとの API 調整や依存関係管理

## レビューフェーズでの協調

### 承認ゲートシステム
- reviewer、chief-product-owner、chief-architect、db-migration による多角的レビュー
- `.claude/desk/outputs/reviews/APPROVALS-ISSUE-<number>.md` での承認管理
- 全員の `approved` 状態確認後の PR 作成

### サブエージェント間通信でのフィードバック
- `.claude/desk/memory/PROTOCOL.md` で定義された通信方式
- CAS ストレージでの成果物共有
- 非同期メッセージングでの効率的な情報交換

## コミュニケーション原則

### 非同期通信の活用
- NDJSON メッセージングでの構造化された情報交換
- queues/outbox システムでの非同期タスク処理
- barriers でのフェーズ同期管理

### 情報品質の向上
- 明確で簡潔な情報共有
- 問題の早期発見と報告
- 建設的なフィードバック
- `.claude/desk/outputs` での体系化された成果物管理

### 知識共有・継続改善
- 各 capabilities でのベストプラクティス共有
- ADR (Architecture Decision Records) での意思決定記録
- プロジェクトを通じた学習・改善サイクル
