# How to use codex-mcp

## 概要

codex-mcpは、OpenAI Codex CLIツールへのブリッジを提供するModel Context Protocol（MCP）サーバーです。Go言語で実装され、JSON-RPC over stdioでMCPプロトコルを実装しています。Claude Codeやその他のMCPクライアントから、Codex CLIの機能を安全かつ構造化された方法で利用できます。

## 機能

- OpenAI Codex CLIへのラッパー機能
- 位置引数でのプロンプト渡し（`codex "prompt"`形式）
- 構造化ログ（JSON/人間可読形式対応）
- タイムアウト制御
- 3つのツール提供：
  - `codex`: プロンプトでのCodex実行
  - `codex_reply`: 追加メッセージ送信
  - `echo`: 接続テスト用

## インストール

### 前提条件

- Go 1.24.6以上
- Codex CLI（事前インストール必要）

### ビルド

```bash
cd mcp/codex-mcp
go build -o bin/codex-mcp
```

または、直接実行：

```bash
go run main.go
```

## 使用例

### MCP設定ファイルに追加

```json
{
  "mcpServers": {
    "codex": {
      "command": "/path/to/codex-mcp/bin/codex-mcp",
      "args": [],
      "env": {
        "MCP_LOG_LEVEL": "info",
        "MCP_LOG_FILE": "/tmp/codex-mcp.log"
      }
    }
  }
}
```

### コマンドライン使用

```bash
# 直接実行（stdioモード）
./bin/codex-mcp

# ログ設定付き実行
MCP_LOG_LEVEL=debug MCP_LOG_JSON=1 ./bin/codex-mcp
```

### MCP Inspector での動作確認

```bash
npx @modelcontextprotocol/inspector --command "./bin/codex-mcp"
```

## 設定オプション

### 環境変数（ログ制御）

- `MCP_LOG_LEVEL`: ログレベル（`debug` | `info`、デフォルト: info）
- `MCP_LOG_JSON`: JSONログ形式を使用（`1` で有効、デフォルト: 人間可読）
- `MCP_LOG_FILE`: ログファイルパス（設定時はstderrと並行出力）
- `MCP_LOG_PARAMS`: パラメータをログに記録（`1` で有効）
- `MCP_LOG_PARAMS_MAX`: パラメータログの最大長（デフォルト: 2048）
- `MCP_REDACT_KEYS`: レダクトするJSONキー（カンマ区切り、例: `api_key,token,password`）

### ツール引数

#### codexツール
- `prompt` (必須): Codexに送るプロンプト
- `cwd` (オプション): 作業ディレクトリ
- `timeoutMs` (オプション): タイムアウト（ミリ秒、デフォルト: 120000）

#### codex_replyツール
- `message` (必須): 追加メッセージ
- `cwd` (オプション): 作業ディレクトリ  
- `timeoutMs` (オプション): タイムアウト

#### echoツール
- `text` (必須): エコーするテキスト

## サーバー情報

- **名前**: codex-mcp-stdio-go
- **バージョン**: 0.2.1-logging
- **プロトコル**: MCP (Model Context Protocol) 2025-06-18互換
- **通信**: JSON-RPC over stdio

## トラブルシューティング

### よくある問題

1. **Codex CLIエラー**: Codex CLIが正しくインストールされているか確認
2. **タイムアウトエラー**: 大きなプロンプトの場合、`timeoutMs`を増やす
3. **権限エラー**: 作業ディレクトリの読み書き権限を確認
4. **ログが見えない**: `MCP_LOG_LEVEL=debug`でデバッグ情報を有効化

### デバッグ

```bash
# 詳細ログで実行
MCP_LOG_LEVEL=debug MCP_LOG_PARAMS=1 MCP_LOG_JSON=1 ./bin/codex-mcp

# ログファイル確認
tail -f /tmp/codex-mcp.log
```
