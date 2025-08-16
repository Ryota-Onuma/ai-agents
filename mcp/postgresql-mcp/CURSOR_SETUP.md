# CursorでのPostgreSQL MCP設定手順

## 概要
このドキュメントでは、CursorでPostgreSQLのMCPツールを使用するための設定手順を説明します。

## 前提条件
- Cursorがインストールされている
- PostgreSQL MCPサーバーが正常に動作している
- PostgreSQLデータベースにアクセスできる

## 設定手順

### 1. MCP設定ファイルの作成
`cursor-mcp-config.json`ファイルを以下の場所にコピーしてください：

**macOSの場合:**
```bash
cp cursor-mcp-config.json ~/Library/Application\ Support/Cursor/User/globalStorage/mcp-config.json
```

**Windowsの場合:**
```bash
copy cursor-mcp-config.json "%APPDATA%\Cursor\User\globalStorage\mcp-config.json"
```

**Linuxの場合:**
```bash
cp cursor-mcp-config.json ~/.config/Cursor/User/globalStorage/mcp-config.json
```

### 2. 環境変数の設定
PostgreSQLに接続するために、以下の環境変数を設定してください：

```bash
# プライマリデータベース
export POSTGRESQL_DEFAULT_URL="postgresql://username:password@localhost:5432/database_name"
export POSTGRESQL_PRIMARY_URL="postgresql://username:password@localhost:5432/database_name"

# セカンダリデータベース（レプリケーション使用時）
export POSTGRESQL_SECONDARY_URL="postgresql://username:password@localhost:5433/database_name"

# 分析用データベース（別途使用時）
export POSTGRESQL_ANALYTICS_URL="postgresql://username:password@localhost:5434/database_name"
```

### 3. Cursorの再起動
設定を反映させるためにCursorを再起動してください。

### 4. MCPツールの確認
Cursorで新しいチャットを開き、以下のプロンプトを試してください：

```
PostgreSQLのMCPツールが利用可能か確認してください。利用可能なツールを一覧表示してください。
```

## 利用可能なツール

以下のツールが利用可能です：

1. **connect_database** - PostgreSQLデータベースに接続
2. **disconnect_database** - データベース接続を切断
3. **list_connections** - アクティブな接続を一覧表示
4. **query** - SELECTクエリを実行
5. **insert** - データを挿入
6. **update** - データを更新
7. **delete** - データを削除
8. **list_schemas** - スキーマを一覧表示
9. **list_tables** - テーブルを一覧表示
10. **describe_table** - テーブルの構造を説明

## トラブルシューティング

### ツールが表示されない場合
1. MCP設定ファイルが正しい場所にあるか確認
2. Cursorを再起動したか確認
3. 環境変数が正しく設定されているか確認

### 接続エラーが発生する場合
1. PostgreSQLサーバーが起動しているか確認
2. 接続情報（ホスト、ポート、ユーザー名、パスワード）が正しいか確認
3. ファイアウォールの設定を確認

## サンプル使用例

```sql
-- データベースに接続
connect_database(name="my_db", connection_string="postgresql://user:pass@localhost:5432/mydb")

-- テーブル一覧を取得
list_tables(schema="public")

-- クエリを実行
query(sql="SELECT * FROM users LIMIT 5")
```

## サポート
問題が発生した場合は、以下を確認してください：
1. MCPサーバーのログ
2. Cursorのコンソール出力
3. 環境変数の設定
