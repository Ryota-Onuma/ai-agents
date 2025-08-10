# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 共通ルール

以下の共通ルールを参照してください：

- [プロジェクト構造](.cursor/rules/project-structure.mdc)

---

## Claude 固有のルール

### サブエージェント活用

- Task tool を使用した専門エージェントの呼び出し
- 複雑なタスクは適切なサブエージェント（planner、architect、reviewer 等）に委譲
- エージェント間の連携とデータの受け渡し最適化

### プロンプト管理

- システムプロンプトの構造化と再利用性
- Few-shot example の効果的な活用
- コンテキスト制限内での情報の最適化

### エージェント設定管理

- .claude/agents/ ディレクトリ内の設定ファイル管理
- Markdown ベースの設定の構造化
- エージェント間の一貫性保持
- プロトコル($PWD/memory/PROTOCOL.md)を用いて、サブエージェント間は通信を行う。

---
