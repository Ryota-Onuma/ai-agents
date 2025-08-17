---
description: 承認された要件定義書に基づいて、技術設計文書を生成する。データフロー図、TypeScriptインターフェース、データベーススキーマ、APIエンドポイントを含む包括的な設計を行います。
---

# design

## 目的

承認された要件定義書に基づいて、技術設計文書を生成する。データフロー図、TypeScript インターフェース、データベーススキーマ、API エンドポイントを含む包括的な設計を行う。

## 使用方法

```bash
design
```

## 前提条件

- `docs/spec/` に要件定義書が存在する
- 要件がユーザによって承認されている
- `CLAUDE.md` が存在し、プロジェクトの技術スタック情報が記載されている
- PostgreSQL MCP が利用可能（DB 関連の要件がある場合）

## 実行内容

**【信頼性レベル指示】**:
各項目について、元の資料（EARS 要件定義書・設計文書含む）との照合状況を以下の信号でコメントしてください：

- 🟢 **青信号**: EARS 要件定義書・設計文書を参考にしてほぼ推測していない場合
- 🟡 **黄信号**: EARS 要件定義書・設計文書から妥当な推測の場合
- 🔴 **赤信号**: EARS 要件定義書・設計文書にない推測の場合

1. **プロジェクトコンテキストの分析**

   - `CLAUDE.md` を読み込み、プロジェクトの技術スタック・アーキテクチャを把握
   - `CLAUDE.md` から辿れる関連ファイル(.cursor/rules/\*)等を確認
   - PostgreSQL MCP を使用して現在のデータベース構造を把握（DB 関連要件がある場合）

2. **要件の分析**

   - `docs/spec/{要件名}-requirements.md` を読み込み
   - 機能要件を整理し、必要な設計要素を特定
   - システムの境界を明確にする
   - 関連する既存設計文書があれば参照する

3. **設計要素の決定**

   - 要件内容に基づいて必要な設計要素を判定：
     - データフロー図（UI/UX 要件がある場合）
     - 型定義ファイル（プログラムロジックがある場合）
     - データベーススキーマ（データ永続化要件がある場合）
     - API 仕様（外部連携・フロントエンド連携がある場合）

4. **アーキテクチャ設計**

   - プロジェクトの技術スタックに基づいたアーキテクチャを決定
   - 既存システムとの整合性を考慮
   - スケーラビリティと保守性を重視

5. **データフロー図の作成**（UI/UX 要件がある場合）

   - Mermaid 記法でデータフローを可視化
   - ユーザーインタラクションの流れ
   - システム間のデータの流れ

6. **型定義ファイルの作成**（プログラムロジックがある場合）

   - プロジェクトの技術スタックに応じた型定義
   - エンティティの型定義
   - API リクエスト/レスポンスの型定義（該当する場合）

7. **データベーススキーマの設計**（データ永続化要件がある場合）

   - 既存 DB スキーマとの整合性を考慮したテーブル定義
   - リレーションシップの設計
   - インデックス戦略

8. **API エンドポイントの設計**（API 要件がある場合）

   - プロジェクトの API 設計パターンに準拠
   - エンドポイントの命名規則
   - リクエスト/レスポンスの構造

9. **ファイルの作成**
   - `docs/design/{要件名}/` ディレクトリに必要なファイルのみ作成：
     - `architecture.md` - アーキテクチャ概要（常に作成）
     - `dataflow.md` - データフロー図（UI/UX 要件がある場合）
     - `types.{ext}` - 型定義（プロジェクトの言語に応じた拡張子）
     - `database-schema.sql` - DB スキーマ（DB 要件がある場合）
     - `api-spec.md` - API 仕様（API 要件がある場合）

## 出力フォーマット例

### architecture.md

```markdown
# {要件名} アーキテクチャ設計

## システム概要

{システムの概要説明}

**関連要件定義**: [📋 {要件名}-requirements.md](../spec/{要件名}-requirements.md)

## プロジェクトコンテキスト

- **技術スタック**: {CLAUDE.md から取得した技術情報}
- **アーキテクチャパターン**: {選択したパターンと理由}
- **既存システムとの関係**: {既存システムとの整合性}

## 設計要素

{要件に応じて必要な設計要素のみリンク表示}

- [📊 データフロー図](dataflow.md) ※UI/UX 要件がある場合
- [🏗️ 型定義](types.{ext}) ※プログラムロジックがある場合
- [🗄️ データベーススキーマ](database-schema.sql) ※DB 要件がある場合
- [🔌 API 仕様](api-spec.md) ※API 要件がある場合

## 実装方針

### {該当する場合のみ記載}

#### フロントエンド

- フレームワーク: {プロジェクトで使用中のフレームワーク}
- 状態管理: {プロジェクトの状態管理方法}

#### バックエンド

- フレームワーク: {プロジェクトで使用中のフレームワーク}
- 認証方式: {プロジェクトの認証方法}

#### データベース

- DBMS: {現在の DB 環境（PostgreSQL MCP から取得）}
- 既存テーブルとの関係: {既存スキーマとの関係性}
```

### dataflow.md

```markdown
# {要件名} データフロー図

## ユーザーインタラクションフロー

\`\`\`mermaid
flowchart TD
A[ユーザー] --> B[フロントエンド]
B --> C[API Gateway]
C --> D[バックエンド]
D --> E[データベース]
\`\`\`

## データ処理フロー

\`\`\`mermaid
sequenceDiagram
participant U as ユーザー
participant F as フロントエンド
participant B as バックエンド
participant D as データベース

    U->>F: アクション
    F->>B: APIリクエスト
    B->>D: クエリ実行
    D-->>B: 結果返却
    B-->>F: レスポンス
    F-->>U: 画面更新

\`\`\`

## 主要な処理フロー

### {機能名 1}のフロー

1. **{ステップ 1}**: {詳細説明}
2. **{ステップ 2}**: {詳細説明}
3. **{ステップ 3}**: {詳細説明}

### {機能名 2}のフロー

{同様の形式で記載}
```

### types.{ext} （プログラムロジックがある場合）

```{language}
// ============================================
// {要件名} 型定義 ({プロジェクトの言語})
// ============================================

// 例：TypeScriptの場合
export interface User {
  id: string;
  email: string;
  name: string;
  createdAt: Date;
  updatedAt: Date;
}

// 例：Pythonの場合（TypedDict使用）
from typing import TypedDict
from datetime import datetime

class User(TypedDict):
    id: str
    email: str
    name: str
    created_at: datetime
    updated_at: datetime

// 例：Goの場合
type User struct {
    ID        string    `json:"id"`
    Email     string    `json:"email"`
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### database-schema.sql （データ永続化要件がある場合）

```sql
-- ============================================
-- {要件名} データベーススキーマ
-- 既存DB構造: {PostgreSQL MCPから取得した現在のスキーマ情報}
-- ============================================

-- 新規テーブル定義
CREATE TABLE {新しいテーブル名} (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- 既存テーブルとの外部キー制約
    existing_table_id UUID REFERENCES existing_table(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- インデックス定義（既存インデックスとの重複を避ける）
CREATE INDEX idx_{table}_{column} ON {table}({column});

-- 既存テーブルの変更（必要に応じて）
ALTER TABLE existing_table ADD COLUMN new_column VARCHAR(255);
```

### api-spec.md （API 要件がある場合）

```markdown
# {要件名} API 仕様

## 概要

{プロジェクトの API 設計パターンに準拠した仕様}

## エンドポイント

### {HTTP_METHOD} {endpoint_path}

**概要**: {エンドポイントの概要}

**リクエスト**:
\`\`\`json
{
"field": "value"
}
\`\`\`

**レスポンス** ({プロジェクトの標準的なレスポンス形式}):
\`\`\`json
{
"success": true,
"data": {
// レスポンスデータ
}
}
\`\`\`

**エラーレスポンス**:
\`\`\`json
{
"success": false,
"error": {
"code": "ERROR_CODE",
"message": "エラーメッセージ"
}
}
\`\`\`

## 認証

{プロジェクトで使用中の認証方式に準拠}

## エラーコード

{プロジェクトの既存エラーコード体系に準拠}
```

## 実行後の確認

- CLAUDE.md からのプロジェクトコンテキスト取得が成功したことを確認
- PostgreSQL MCP からの DB 情報取得が成功したことを確認（DB 要件がある場合）
- 要件定義書の読み込みが成功したことを確認
- 作成したファイルの一覧を表示（要件に応じて必要なファイルのみ）
  - `docs/design/{要件名}/architecture.md` （常に作成）
  - `docs/design/{要件名}/dataflow.md` （UI/UX 要件がある場合）
  - `docs/design/{要件名}/types.{ext}` （プログラムロジックがある場合）
  - `docs/design/{要件名}/database-schema.sql` （DB 要件がある場合）
  - `docs/design/{要件名}/api-spec.md` （API 要件がある場合）
- 設計の主要なポイントをサマリーで表示
- プロジェクトの技術スタックとの整合性を確認
- 既存システムとの関係性を確認
- ユーザに確認を促すメッセージを表示
