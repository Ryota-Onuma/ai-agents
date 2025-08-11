// main.go
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"
)

/*
  改築ポイント（要約）
  - stdout は MCP JSON-RPC 専用（1行1JSON）。ログは stderr/任意ログファイルへ。
  - リク/レスを構造化ログ（method,id,duration,tool,isError,outLen等）。
  - 環境変数でログ制御：
      MCP_LOG_JSON=1            : JSONログ（デフォルトは人間可読）
      MCP_LOG_FILE=/path/to.log : ログファイルにも追記
      MCP_LOG_LEVEL=debug|info  : ログレベル
      MCP_LOG_PARAMS=1          : params/argumentsの一部を記録
      MCP_LOG_PARAMS_MAX=2048   : params最大長
      MCP_REDACT_KEYS=api_key,token,password : JSONキーをレダクト
  - `codex`/`codex_reply` は「--prompt」をやめ、**位置引数**で実行（例: codex "text"）。
*/

//
// ------------------------- JSON-RPC 型 -------------------------
//

type jsonrpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // number | string | null
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcRes struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

//
// ------------------------- ログ基盤 -------------------------
//

type logLevel int

const (
	levelDebug logLevel = iota
	levelInfo
)

type logger struct {
	jsonMode     bool
	level        logLevel
	writers      []io.Writer
	mu           sync.Mutex
	paramsLog    bool
	paramsMaxLen int
	redactRx     []*regexp.Regexp
}

func newLoggerFromEnv() *logger {
	level := levelInfo
	switch strings.ToLower(os.Getenv("MCP_LOG_LEVEL")) {
	case "debug":
		level = levelDebug
	}
	jsonMode := os.Getenv("MCP_LOG_JSON") == "1"
	paramsLog := os.Getenv("MCP_LOG_PARAMS") == "1"
	maxLen := 2048
	if v := strings.TrimSpace(os.Getenv("MCP_LOG_PARAMS_MAX")); v != "" {
		if n, err := fmt.Sscanf(v, "%d", &maxLen); n == 1 && err == nil && maxLen > 0 {
			// ok
		}
	}

	var writers []io.Writer
	writers = append(writers, os.Stderr)
	if p := os.Getenv("MCP_LOG_FILE"); p != "" {
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		if f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err == nil {
			writers = append(writers, f)
		} else {
			fmt.Fprintf(os.Stderr, "[log-open-error] %v\n", err)
		}
	}

	redactKeys := strings.Split(strings.ToLower(os.Getenv("MCP_REDACT_KEYS")), ",")
	var rx []*regexp.Regexp
	for _, k := range redactKeys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		rx = append(rx, regexp.MustCompile(`(?i)("`+regexp.QuoteMeta(k)+`"\s*:\s*)(".*?"|'.*?'|[^\s,}\]]+)`))
	}

	return &logger{
		jsonMode:     jsonMode,
		level:        level,
		writers:      writers,
		paramsLog:    paramsLog,
		paramsMaxLen: maxLen,
		redactRx:     rx,
	}
}

func (l *logger) Debug(evt string, fields map[string]any) { l.log(levelDebug, evt, fields) }
func (l *logger) Info(evt string, fields map[string]any)  { l.log(levelInfo, evt, fields) }

func (l *logger) log(lv logLevel, evt string, fields map[string]any) {
	if lv < l.level {
		return
	}
	if fields == nil {
		fields = map[string]any{}
	}
	fields["ts"] = time.Now().Format(time.RFC3339Nano)
	fields["event"] = evt
	fields["level"] = map[logLevel]string{levelDebug: "debug", levelInfo: "info"}[lv]

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.jsonMode {
		b, _ := json.Marshal(fields)
		for _, w := range l.writers {
			fmt.Fprintln(w, string(b))
		}
		return
	}

	// human-readable
	var b strings.Builder
	b.WriteString(fmt.Sprintf("[%s] %s", fields["ts"], evt))
	if lvl, ok := fields["level"].(string); ok {
		b.WriteString(" (" + lvl + ")")
	}
	delete(fields, "ts")
	delete(fields, "event")
	delete(fields, "level")
	for k, v := range fields {
		b.WriteString(fmt.Sprintf(" %s=%v", k, v))
	}
	line := b.String()
	for _, w := range l.writers {
		fmt.Fprintln(w, line)
	}
}

func (l *logger) maybeParams(raw json.RawMessage) string {
	if !l.paramsLog || len(raw) == 0 {
		return ""
	}
	s := string(raw)
	for _, r := range l.redactRx {
		s = r.ReplaceAllString(s, `${1}"[REDACTED]"`)
	}
	if len(s) > l.paramsMaxLen {
		return s[:l.paramsMaxLen] + "...(truncated)"
	}
	return s
}

//
// ------------------------- MCP応答ユーティリティ -------------------------
//

func writeResult(id json.RawMessage, result any) {
	res := jsonrpcRes{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	encWrite(res)
}

func writeError(id json.RawMessage, code int, msg string, data any) {
	res := jsonrpcRes{
		JSONRPC: "2.0",
		ID:      id,
		Error: &jsonrpcError{
			Code:    code,
			Message: msg,
			Data:    data,
		},
	}
	encWrite(res)
}

func encWrite(v any) {
	// IMPORTANT: stdoutはMCPメッセージ専用（1行1JSON）。ログはstderrへ。
	b, err := json.Marshal(v)
	if err != nil {
		logg.Debug("marshal_error", map[string]any{"error": err.Error()})
		return
	}
	os.Stdout.Write(b)
	os.Stdout.Write([]byte("\n"))
}

//
// ------------------------- MCP payloads -------------------------
//

type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type initializeResult struct {
	ProtocolVersion string `json:"protocolVersion"`
	Capabilities    any    `json:"capabilities"`
	ServerInfo      any    `json:"serverInfo"`
	Instructions    string `json:"instructions"`
}

type toolsListResult struct {
	Tools      []toolDef `json:"tools"`
	NextCursor any       `json:"nextCursor"` // null
}

type toolDef struct {
	Name        string         `json:"name"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type toolCallResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError"`
}

type toolContent struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

//
// ------------------------- サーバー定義 -------------------------
//

var serverInfo = map[string]any{
	"name":    "codex-mcp-stdio-go",
	"title":   "Codex MCP (stdio) Go",
	"version": "0.2.1-logging",
}

var tools = []toolDef{
	{
		Name:        "codex",
		Title:       "Run Codex with a prompt",
		Description: "Run OpenAI Codex CLI with a prompt string (interactive TUI).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt": map[string]any{"type": "string", "description": "Prompt for codex CLI"},
				"cwd":    map[string]any{"type": "string", "description": "Working directory (optional)"},
				"timeoutMs": map[string]any{
					"type":        "number",
					"description": "Optional timeout in milliseconds (default 120000)",
				},
			},
			"required": []string{"prompt"},
		},
	},
	{
		Name:        "codex_reply",
		Title:       "Reply to previous Codex session",
		Description: "Send a follow-up message to Codex (simple prompt passthrough).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string", "description": "Follow-up message"},
				"cwd":     map[string]any{"type": "string", "description": "Working directory (optional)"},
				"timeoutMs": map[string]any{
					"type":        "number",
					"description": "Optional timeout in milliseconds (default 120000)",
				},
			},
			"required": []string{"message"},
		},
	},
	// 接続テスト＆ログ確認用
	{
		Name:        "echo",
		Title:       "Echo text (for connectivity/logging test)",
		Description: "Echo back given text",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"text": map[string]any{"type": "string"},
			},
			"required": []string{"text"},
		},
	},
}

//
// ------------------------- Codex 実行 -------------------------
//

type runOpts struct {
	Cwd       string
	TimeoutMs int
}

func runCodex(args []string, opt runOpts) (string, error) {
	// 位置引数でプロンプトを渡す（例: codex "text"）
	cmd := exec.Command("codex", args...)
	if opt.Cwd != "" {
		cmd.Dir = opt.Cwd
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if opt.TimeoutMs <= 0 {
		opt.TimeoutMs = 120_000 // 120s default
	}

	if err := cmd.Start(); err != nil {
		return "", err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			if errBuf.Len() > 0 {
				return "", fmt.Errorf("codex: %w: %s", err, errBuf.String())
			}
			return "", fmt.Errorf("codex: %w", err)
		}
		out := outBuf.String()
		if len(out) == 0 {
			out = "(no output)"
		}
		return out, nil
	case <-time.After(time.Duration(opt.TimeoutMs) * time.Millisecond):
		_ = cmd.Process.Kill()
		return "", errors.New("codex: timeout")
	}
}

//
// ------------------------- ディスパッチャ -------------------------
//

var logg = newLoggerFromEnv()

func handle(ctx context.Context, req jsonrpcReq) {
	start := time.Now()
	reqID := strings.TrimSpace(string(req.ID))

	// リクエスト受信ログ
	logg.Info("rpc_request", map[string]any{
		"id":     reqID,
		"method": req.Method,
		"params": logg.maybeParams(req.Params),
	})

	switch req.Method {
	case "initialize":
		var p initializeParams
		if len(req.Params) > 0 {
			_ = json.Unmarshal(req.Params, &p)
		}
		if p.ProtocolVersion == "" {
			p.ProtocolVersion = "2025-06-18"
		}
		logg.Info("mcp_initialize", map[string]any{
			"id":              reqID,
			"protocolVersion": p.ProtocolVersion,
			"serverVersion":   serverInfo["version"],
		})
		writeResult(req.ID, initializeResult{
			ProtocolVersion: p.ProtocolVersion,
			Capabilities: map[string]any{
				"logging": map[string]any{},
				"tools":   map[string]any{"listChanged": true},
			},
			ServerInfo:   serverInfo,
			Instructions: "Codex CLI bridge via MCP stdio (Go).",
		})
		logDuration("rpc_response", start, map[string]any{"id": reqID, "method": req.Method})
		return

	case "notifications/initialized":
		logg.Info("client_initialized", map[string]any{"id": reqID})
		return

	case "tools/list":
		writeResult(req.ID, toolsListResult{Tools: tools, NextCursor: nil})
		logDuration("rpc_response", start, map[string]any{
			"id": reqID, "method": req.Method, "toolsCount": len(tools),
		})
		return

	case "tools/call":
		var p toolsCallParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError(req.ID, -32602, "invalid params", err.Error())
			logDuration("rpc_error", start, map[string]any{"id": reqID, "method": req.Method, "error": "invalid params"})
			return
		}
		logg.Info("tool_dispatch", map[string]any{
			"id": reqID, "tool": p.Name, "arguments": logg.maybeParams(p.Arguments),
		})

		switch p.Name {
		case "codex":
			var a struct {
				Prompt    string `json:"prompt"`
				Cwd       string `json:"cwd"`
				TimeoutMs int    `json:"timeoutMs"`
			}
			if err := json.Unmarshal(p.Arguments, &a); err != nil {
				writeError(req.ID, -32602, "invalid arguments", err.Error())
				logDuration("rpc_error", start, map[string]any{"id": reqID, "method": req.Method, "tool": p.Name, "error": "invalid arguments"})
				return
			}
			if a.Prompt == "" {
				writeError(req.ID, -32602, "prompt is required", nil)
				logDuration("rpc_error", start, map[string]any{"id": reqID, "method": req.Method, "tool": p.Name, "error": "prompt required"})
				return
			}
			// 位置引数で渡す（codex "..."）
			out, err := runCodex([]string{a.Prompt}, runOpts{Cwd: a.Cwd, TimeoutMs: a.TimeoutMs})
			if err != nil {
				writeResult(req.ID, toolCallResult{Content: []toolContent{{Type: "text", Text: err.Error()}}, IsError: true})
				logDuration("tool_result", start, map[string]any{"id": reqID, "tool": p.Name, "isError": true, "err": err.Error()})
				return
			}
			writeResult(req.ID, toolCallResult{Content: []toolContent{{Type: "text", Text: out}}, IsError: false})
			logDuration("tool_result", start, map[string]any{"id": reqID, "tool": p.Name, "isError": false, "outLen": len(out)})
			return

		case "codex_reply":
			var a struct {
				Message   string `json:"message"`
				Cwd       string `json:"cwd"`
				TimeoutMs int    `json:"timeoutMs"`
			}
			if err := json.Unmarshal(p.Arguments, &a); err != nil {
				writeError(req.ID, -32602, "invalid arguments", err.Error())
				logDuration("rpc_error", start, map[string]any{"id": reqID, "method": req.Method, "tool": p.Name, "error": "invalid arguments"})
				return
			}
			if a.Message == "" {
				writeError(req.ID, -32602, "message is required", nil)
				logDuration("rpc_error", start, map[string]any{"id": reqID, "method": req.Method, "tool": p.Name, "error": "message required"})
				return
			}
			// 位置引数で渡す（codex "..."）
			out, err := runCodex([]string{a.Message}, runOpts{Cwd: a.Cwd, TimeoutMs: a.TimeoutMs})
			if err != nil {
				writeResult(req.ID, toolCallResult{Content: []toolContent{{Type: "text", Text: err.Error()}}, IsError: true})
				logDuration("tool_result", start, map[string]any{"id": reqID, "tool": p.Name, "isError": true, "err": err.Error()})
				return
			}
			writeResult(req.ID, toolCallResult{Content: []toolContent{{Type: "text", Text: out}}, IsError: false})
			logDuration("tool_result", start, map[string]any{"id": reqID, "tool": p.Name, "isError": false, "outLen": len(out)})
			return

		case "echo":
			var a struct {
				Text string `json:"text"`
			}
			if err := json.Unmarshal(p.Arguments, &a); err != nil {
				writeError(req.ID, -32602, "invalid arguments", err.Error())
				logDuration("rpc_error", start, map[string]any{"id": reqID, "method": req.Method, "tool": p.Name, "error": "invalid arguments"})
				return
			}
			writeResult(req.ID, toolCallResult{Content: []toolContent{{Type: "text", Text: a.Text}}, IsError: false})
			logDuration("tool_result", start, map[string]any{"id": reqID, "tool": p.Name, "isError": false, "outLen": len(a.Text)})
			return

		default:
			writeError(req.ID, -32602, "unknown tool: "+p.Name, nil)
			logDuration("rpc_error", start, map[string]any{"id": reqID, "method": req.Method, "tool": p.Name, "error": "unknown tool"})
			return
		}

	case "ping":
		writeResult(req.ID, map[string]any{"ok": true, "now": time.Now().UTC().Format(time.RFC3339)})
		logDuration("rpc_response", start, map[string]any{"id": reqID, "method": req.Method})
		return

	default:
		writeError(req.ID, -32601, "Method not found: "+req.Method, nil)
		logDuration("rpc_error", start, map[string]any{"id": reqID, "method": req.Method, "error": "method not found"})
		return
	}
}

func logDuration(evt string, start time.Time, base map[string]any) {
	if base == nil {
		base = map[string]any{}
	}
	base["durationMs"] = time.Since(start).Milliseconds()
	logg.Info(evt, base)
}

//
// ------------------------- main ループ -------------------------
//

func main() {
	logg.Info("server_start", map[string]any{
		"server": serverInfo["name"],
		"ver":    serverInfo["version"],
		"pid":    os.Getpid(),
	})

	// SIGINT/SIGTERM でクリーン終了
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		ch := make(chan os.Signal, 2)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		s := <-ch
		logg.Info("server_signal", map[string]any{"signal": s.String()})
		cancel()
	}()

	reader := bufio.NewReader(os.Stdin)
	sc := bufio.NewScanner(reader)
	buf := make([]byte, 0, 1024*1024)
	sc.Buffer(buf, 16*1024*1024)

	for {
		select {
		case <-ctx.Done():
			logg.Info("server_stop", map[string]any{"reason": "signal"})
			return
		default:
		}

		if !sc.Scan() {
			if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
				logg.Info("scan_error", map[string]any{"error": err.Error()})
			}
			logg.Info("server_stop", map[string]any{"reason": "eof"})
			break
		}
		line := sc.Text()
		if len(line) == 0 {
			continue
		}
		var req jsonrpcReq
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			logg.Info("parse_error", map[string]any{"error": err.Error()})
			continue
		}
		handle(ctx, req)
	}
}
