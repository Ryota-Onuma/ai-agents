## サブエージェント間通信プロトコル（desk/memory）

- サブエージェント同士の直接同期通信は不可。**`desk/memory/` ディレクトリを介した非同期通信**を標準とする。
- **エンベロープ形式**は `queues/*.inbox.ndjson`（受信キュー）と `outbox/*.outbox.ndjson`（送信キュー）を使用する。
- **CAS ストレージ**：大きな成果物は `cas/sha256/<2桁>/<2桁>/<hash>.<ext>` に格納し、メッセージの `attachments.hash` フィールドで参照する。
- **バリアファイル**：フェーズごとの同期ポイントは `barriers/phase-<n>.barrier` で管理する。削除されたら次フェーズに進める。
- **ロックファイル**：`locks/<resource>.lock` は原子的 rename で取得・解放する。

### 運用ルール

1. **タスク送信**

   - `to` に宛先エージェントを指定し、対応する `queues/<agent>.inbox.ndjson` に追記する。
   - 必要な成果物は先に `cas/` に保存し、`attachments` に参照を付与。

2. **タスク受信**

   - 自分宛の `inbox` を読み、未処理タスクを順次処理する。
   - 処理結果は `outbox/<self>.outbox.ndjson` に書き込む。

3. **完了通知（ack）**

   - 親エージェントは `receipts/` に ack ファイルを作成し、再送防止と監査に利用する。

4. **同期ポイント**
   - フェーズ移行時に `barriers/` を監視。削除時に次ステップを開始する。

### NDJSON メッセージ雛形

```json
{
  "id": "01J...",
  "ts": "2025-08-10T01:23:45+09:00",
  "from": "planner",
  "to": "architect",
  "type": "TASK",
  "corr": "01J...",
  "ttlSec": 7200,
  "priority": 5,
  "spec": {
    "task": "design_api",
    "inputs": { "issue": "https://.../issues/123" }
  },
  "attachments": [
    { "hash": "sha256:ab...", "mime": "text/markdown", "rel": "spec" }
  ]
}
```
