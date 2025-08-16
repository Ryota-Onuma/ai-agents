# PostgreSQL MCP Server Setup Guide

Claude CodeとCursorからPostgreSQL MCPサーバーに接続するための設定手順です。

## 前提条件

- Python 3.8以上
- Docker & Docker Compose
- Claude Code または Cursor

## 1. 依存関係のインストール

```bash
cd mcp/postgresql-mcp
pip install -r requirements.txt
```

## 2. PostgreSQL データベースの起動

```bash
# Docker Composeでデータベースを起動
docker-compose -f docker-compose.example.yml up -d

# 起動確認
docker-compose -f docker-compose.example.yml ps
```

これで以下のデータベースが利用可能になります：
- **Primary DB**: localhost:5432 (primary_db)
- **Secondary DB**: localhost:5433 (secondary_db) 
- **Analytics DB**: localhost:5434 (analytics_db)
- **pgAdmin**: localhost:8080 (管理画面)

## 3. 環境設定

```bash
# 環境変数ファイルをコピー
cp .env.example .env

# 必要に応じて接続情報を編集
nano .env
```

## 4. 接続テスト

```bash
# MCPサーバーの動作テスト
python test-connection.py
```

## 5. Claude Code での設定

### 方法1: 設定ファイルを直接コピー

```bash
# Claude Code の設定ディレクトリに移動
cd ~/.config/claude-code  # Linux/Mac
# または
cd %APPDATA%\claude-code  # Windows

# 設定をマージ
cat /path/to/mcp/postgresql-mcp/claude-code-config.json >> mcp_settings.json
```

### 方法2: 手動設定

Claude Code の MCP 設定に以下を追加：

```json
{
  "mcpServers": {
    "postgresql": {
      "command": "python3",
      "args": ["/Users/ryota/Desktop/programming/ai-agents/mcp/postgresql-mcp/main.py"],
      "env": {
        "POSTGRESQL_DEFAULT_URL": "postgresql://postgres:password123@localhost:5432/primary_db",
        "POSTGRESQL_PRIMARY_URL": "postgresql://postgres:password123@localhost:5432/primary_db",
        "POSTGRESQL_SECONDARY_URL": "postgresql://postgres:password123@localhost:5433/secondary_db",
        "POSTGRESQL_ANALYTICS_URL": "postgresql://postgres:password123@localhost:5434/analytics_db"
      }
    }
  }
}
```

### パス設定の注意点

**重要**: `args` の配列内のパスは絶対パスで指定してください：

```json
"args": ["/Users/ryota/Desktop/programming/ai-agents/mcp/postgresql-mcp/main.py"]
```

相対パスや `~` は使用できません。

## 6. Cursor での設定

### Cursor の設定ファイル場所

- **Mac**: `~/Library/Application Support/Cursor/User/settings.json`
- **Windows**: `%APPDATA%\Cursor\User\settings.json`
- **Linux**: `~/.config/Cursor/User/settings.json`

### 設定内容

```json
{
  "mcp": {
    "servers": {
      "postgresql": {
        "command": "python3",
        "args": ["/Users/ryota/Desktop/programming/ai-agents/mcp/postgresql-mcp/main.py"],
        "env": {
          "POSTGRESQL_DEFAULT_URL": "postgresql://postgres:password123@localhost:5432/primary_db",
          "POSTGRESQL_PRIMARY_URL": "postgresql://postgres:password123@localhost:5432/primary_db",
          "POSTGRESQL_SECONDARY_URL": "postgresql://postgres:password123@localhost:5433/secondary_db",
          "POSTGRESQL_ANALYTICS_URL": "postgresql://postgres:password123@localhost:5434/analytics_db"
        }
      }
    }
  }
}
```

## 7. 使用方法

### 基本的なコマンド

```bash
# 接続一覧を確認
list_connections

# テーブル一覧を取得
list_tables

# 特定のデータベースのテーブル一覧
list_tables database="secondary"

# データを検索
query sql="SELECT * FROM users LIMIT 5"

# 特定のデータベースで検索
query sql="SELECT * FROM inventory LIMIT 5" database="secondary"

# データを挿入
insert table="users" data='{"username": "alice", "email": "alice@example.com"}'

# データを更新
update table="users" data='{"email": "alice.new@example.com"}' where='{"id": 1}'

# テーブル構造を確認
describe_table table="users"
```

### 新しいデータベースに接続

```bash
# 新しいデータベース接続を追加
connect_database name="my_app" connection_string="postgresql://user:pass@host:port/database"

# 接続したデータベースを使用
query sql="SELECT version()" database="my_app"
```

## 8. トラブルシューティング

### よくある問題

#### 1. "No module named 'asyncpg'" エラー
```bash
pip install -r requirements.txt
```

#### 2. データベース接続エラー
- Docker コンテナが起動しているか確認
- ポート番号が正しいか確認
- パスワードが正しいか確認

```bash
# コンテナ状態確認
docker-compose -f docker-compose.example.yml ps

# ログ確認
docker-compose -f docker-compose.example.yml logs postgres-primary
```

#### 3. MCP サーバーが認識されない
- パスが絶対パスになっているか確認
- Python 環境が正しいか確認
- ファイルが実行可能か確認

```bash
# パスの確認
which python3
ls -la /Users/ryota/Desktop/programming/ai-agents/mcp/postgresql-mcp/main.py
```

#### 4. 権限エラー
```bash
# 実行権限を付与
chmod +x /Users/ryota/Desktop/programming/ai-agents/mcp/postgresql-mcp/main.py
chmod +x /Users/ryota/Desktop/programming/ai-agents/mcp/postgresql-mcp/run.sh
```

### デバッグ用コマンド

```bash
# MCP サーバーを直接実行してデバッグ
cd /Users/ryota/Desktop/programming/ai-agents/mcp/postgresql-mcp
python main.py

# 詳細なログを有効にして実行
MCP_LOG_LEVEL=debug python main.py

# 接続テストを実行
python test-connection.py
```

## 9. カスタム設定

### 独自のデータベースを追加

`.env` ファイルに新しい接続を追加：

```bash
POSTGRESQL_MYSERVICE_URL=postgresql://user:pass@host:port/myservice_db
```

環境変数は自動的に読み込まれ、`POSTGRESQL_*_URL` パターンのものは接続として登録されます。

### セキュリティ設定

本番環境では以下を考慮：

```bash
# SSL接続の有効化
POSTGRESQL_DEFAULT_URL=postgresql://user:pass@host:port/db?sslmode=require

# 接続プールの設定
DB_POOL_MIN_SIZE=1
DB_POOL_MAX_SIZE=10
DB_POOL_TIMEOUT=30
```

## 10. 次のステップ

1. Claude Code または Cursor を再起動
2. PostgreSQL ツールが利用可能になっているか確認
3. 実際のクエリを試してみる
4. 独自のデータベーススキーマに合わせてカスタマイズ

## サポート

問題が発生した場合：

1. `test-connection.py` を実行して基本的な動作確認
2. Docker コンテナのログを確認
3. MCP サーバーログを確認
4. 設定ファイルのパスと権限を確認

---

**注意**: このセットアップは開発環境用です。本番環境では適切なセキュリティ設定を行ってください。