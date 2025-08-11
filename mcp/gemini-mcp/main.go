// main.go
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

/*
  Gemini MCP (stdio) Go — one-shot safe wrapper for the official Gemini CLI.

  目的:
    - 「一発で正しく」gemini CLI を使わせる。必ず `-p/--prompt` を付けて実行。
    - モデル指定やデバッグ等の主要オプションだけを明示的に許可（誤用を減らす）。

  このサーバが行うこと:
    - tools:
        1) gemini        : 非対話・一発実行（必ず -p を付与）
        2) gemini_reply  : 追伸（実体は gemini と同じ、引数名が message）
        3) gemini_raw    : 上級者向けの素通し（必要時のみ）
    - stdout は 1 行 1 JSON（JSON-RPC 2.0）。ログは stderr。

  重要条件・数値:
    - デフォルトタイムアウト: 120000 ms（環境に合わせて runOpts.TimeoutMs で調整）
    - プロトコルバージョン: 2025-06-18 互換（echo系と合わせるための暫定）
    - 非対話実行は `gemini -p "<prompt>"` を強制。ExtraArgs 内の `-p/--prompt` は拒否。
    - モデル指定 `-m/--model` は任意。未指定時は CLI 既定（現状: 2.5 Pro が既定、混雑時に Flash へフォールバックする実装あり）

  仕様確認（取得日: 2025-08-10 JST）:
    - 非対話実行フラグ `-p/--prompt` の存在: Firebase / Docs サイトで明記
    - 既定モデルと `-m/--model` 指定: 解説記事で明示（2.5 Pro 既定、状況により Flash フォールバック）
    - デバッグ `-d/--debug`: Issue で実例
    - 全ファイル含めるショートカット `-a`（long 名は実装により `--all_files` / `--all-files` / `--include-all-files` など揺れあり。本ブリッジは確実性重視で短縮 `-a` を採用）
    - YOLO（自動承認）`--yolo`: ヘルプ断片・Issueに登場

  URL（参考・一次情報優先／日付はページ内参照）:
    - Non-interactive `-p/--prompt` の記述（公式ドキュメント系）:
      https://gemini-cli.xyz/docs/en/cli/        // Non-interactive mode に -p 記載
    - 既定モデル/モデル切替（実測レビュー・最新動作の把握用）:
      https://www.infoworld.com/article/4025916/ai-coding-at-the-command-line-with-gemini-cli.html
    - `--debug` フラグの実例（GitHub Issues）:
      https://github.com/google-gemini/gemini-cli/issues/4661
      https://github.com/google-gemini/gemini-cli/issues/4042
    - 全ファイル含める系フラグの実例（ヘルプ断片・解説）:
      https://medium.com/google-cloud/gemini-cli-tutorial-series-part-2-gemini-cli-command-line-parameters-e64e21b157be
      https://wietsevenema.eu/blog/2025/how-gemini-cli-builds-context/
*/

type jsonrpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
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

// ---------- write helpers ----------
func writeResult(id json.RawMessage, result any) {
	encWrite(jsonrpcRes{JSONRPC: "2.0", ID: id, Result: result})
}
func writeError(id json.RawMessage, code int, msg string, data any) {
	encWrite(jsonrpcRes{
		JSONRPC: "2.0", ID: id,
		Error: &jsonrpcError{Code: code, Message: msg, Data: data},
	})
}
func encWrite(v any) {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[marshal-error] %v\n", err)
		return
	}
	// IMPORTANT: stdout は 1 行 1 JSON（MCP 用）。ログは stderr。
	_, _ = os.Stdout.Write(b)
	_, _ = os.Stdout.Write([]byte("\n"))
}

// ---------- MCP payloads ----------
type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}
type initializeResult struct {
	ProtocolVersion string `json:"protocolVersion"`
	Capabilities    any    `json:"capabilities"`
	ServerInfo      any    `json:"serverInfo"`
	Instructions    string `json:"instructions"`
}

type toolDef struct {
	Name        string         `json:"name"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}
type toolsListResult struct {
	Tools      []toolDef `json:"tools"`
	NextCursor any       `json:"nextCursor"`
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

// ---------- server static ----------
var serverInfo = map[string]any{
	"name":    "gemini-mcp-stdio-go",
	"title":   "Gemini MCP (stdio) Go",
	"version": "0.2.0",
}

var tools = []toolDef{
	{
		Name:        "gemini",
		Title:       "Run Gemini with a prompt (non-interactive, always -p)",
		Description: "Executes the Gemini CLI once using -p/--prompt with your text.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"prompt": map[string]any{"type": "string", "description": "Prompt passed to gemini -p (required)"},
				"cwd":    map[string]any{"type": "string", "description": "Working directory (optional)"},
				"timeoutMs": map[string]any{
					"type":        "number",
					"description": "Timeout in milliseconds (default 120000)",
				},
				// 主要オプション（必要最小限）
				"model": map[string]any{"type": "string", "description": "e.g. gemini-2.5-pro"},
				"debug": map[string]any{"type": "boolean", "description": "add -d/--debug"},
				"all":   map[string]any{"type": "boolean", "description": "add -a (include ALL files as context)"},
				"yolo":  map[string]any{"type": "boolean", "description": "add --yolo (auto-accept actions)"},
				"extraArgs": map[string]any{
					"type":        "array",
					"items":       map[string]any{"type": "string"},
					"description": "Additional raw args (no -p/--prompt here)",
				},
			},
			"required": []string{"prompt"},
		},
	},
	{
		Name:        "gemini_reply",
		Title:       "Send a follow-up to Gemini (one-shot, always -p)",
		Description: "Same as gemini tool but uses 'message' field for clarity.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{"type": "string"},
				"cwd":     map[string]any{"type": "string"},
				"timeoutMs": map[string]any{
					"type":        "number",
					"description": "Timeout in milliseconds (default 120000)",
				},
				"model":     map[string]any{"type": "string"},
				"debug":     map[string]any{"type": "boolean"},
				"all":       map[string]any{"type": "boolean"},
				"yolo":      map[string]any{"type": "boolean"},
				"extraArgs": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"message"},
		},
	},
	{
		Name:        "gemini_raw",
		Title:       "Raw Gemini CLI passthrough (advanced)",
		Description: "Runs `gemini` with an arbitrary args array. Prefer 'gemini' for typical usage.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"args":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Args passed directly to `gemini`"},
				"cwd":       map[string]any{"type": "string"},
				"timeoutMs": map[string]any{"type": "number"},
			},
			"required": []string{"args"},
		},
	},
}

// ---------- gemini exec ----------
type runOpts struct {
	Cwd       string
	TimeoutMs int
}

func runGemini(args []string, opt runOpts) (string, error) {
	cmd := exec.Command("gemini", args...) // IMPORTANT: no shell
	if opt.Cwd != "" {
		cmd.Dir = opt.Cwd
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	if opt.TimeoutMs <= 0 {
		opt.TimeoutMs = 120_000
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
				return "", fmt.Errorf("gemini: %w: %s", err, errBuf.String())
			}
			return "", fmt.Errorf("gemini: %w", err)
		}
		out := outBuf.String()
		if len(out) == 0 {
			out = "(no output)"
		}
		return out, nil
	case <-time.After(time.Duration(opt.TimeoutMs) * time.Millisecond):
		_ = cmd.Process.Kill()
		return "", errors.New("gemini: timeout")
	}
}

func denyIfHasPromptFlag(xs []string) error {
	for _, a := range xs {
		switch a {
		case "-p", "--prompt":
			return errors.New("extraArgs must NOT contain -p/--prompt; this server always supplies -p internally")
		}
	}
	return nil
}

// ---------- dispatcher ----------
func handle(req jsonrpcReq) {
	switch req.Method {
	case "initialize":
		var p initializeParams
		_ = json.Unmarshal(req.Params, &p)
		if p.ProtocolVersion == "" {
			p.ProtocolVersion = "2025-06-18"
		}
		writeResult(req.ID, initializeResult{
			ProtocolVersion: p.ProtocolVersion,
			Capabilities: map[string]any{
				"logging": map[string]any{},
				"tools":   map[string]any{"listChanged": true},
			},
			ServerInfo: serverInfo,
			Instructions: "Gemini CLI bridge via MCP stdio (Go). " +
				"Non-interactive runs always use -p. Provide `prompt` (or `message`) and optional `model`, `debug`, `all`, `yolo`.",
		})
		return

	case "notifications/initialized":
		return // no response

	case "tools/list":
		writeResult(req.ID, toolsListResult{Tools: tools, NextCursor: nil})
		return

	case "tools/call":
		var p toolsCallParams
		if err := json.Unmarshal(req.Params, &p); err != nil {
			writeError(req.ID, -32602, "invalid params", err.Error())
			return
		}
		switch p.Name {

		case "gemini":
			var a struct {
				Prompt    string   `json:"prompt"`
				Cwd       string   `json:"cwd"`
				TimeoutMs int      `json:"timeoutMs"`
				Model     string   `json:"model"`
				Debug     bool     `json:"debug"`
				All       bool     `json:"all"`
				Yolo      bool     `json:"yolo"`
				ExtraArgs []string `json:"extraArgs"`
			}
			if err := json.Unmarshal(p.Arguments, &a); err != nil {
				writeError(req.ID, -32602, "invalid arguments", err.Error())
				return
			}
			if a.Prompt == "" {
				writeError(req.ID, -32602, "prompt is required", nil)
				return
			}
			if err := denyIfHasPromptFlag(a.ExtraArgs); err != nil {
				writeError(req.ID, -32602, err.Error(), nil)
				return
			}

			args := []string{"-p", a.Prompt} // 一発実行の基本形（強制）
			if a.Model != "" {
				args = append(args, "-m", a.Model)
			}
			if a.Debug {
				args = append(args, "-d")
			}
			if a.All {
				// long 名は実装で揺れるためショートの -a を採用
				args = append(args, "-a")
			}
			if a.Yolo {
				args = append(args, "--yolo")
			}
			args = append(args, a.ExtraArgs...)

			out, err := runGemini(args, runOpts{Cwd: a.Cwd, TimeoutMs: a.TimeoutMs})
			if err != nil {
				writeResult(req.ID, toolCallResult{Content: []toolContent{{Type: "text", Text: err.Error()}}, IsError: true})
				return
			}
			writeResult(req.ID, toolCallResult{Content: []toolContent{{Type: "text", Text: out}}, IsError: false})
			return

		case "gemini_reply":
			var a struct {
				Message   string   `json:"message"`
				Cwd       string   `json:"cwd"`
				TimeoutMs int      `json:"timeoutMs"`
				Model     string   `json:"model"`
				Debug     bool     `json:"debug"`
				All       bool     `json:"all"`
				Yolo      bool     `json:"yolo"`
				ExtraArgs []string `json:"extraArgs"`
			}
			if err := json.Unmarshal(p.Arguments, &a); err != nil {
				writeError(req.ID, -32602, "invalid arguments", err.Error())
				return
			}
			if a.Message == "" {
				writeError(req.ID, -32602, "message is required", nil)
				return
			}
			if err := denyIfHasPromptFlag(a.ExtraArgs); err != nil {
				writeError(req.ID, -32602, err.Error(), nil)
				return
			}

			args := []string{"-p", a.Message}
			if a.Model != "" {
				args = append(args, "-m", a.Model)
			}
			if a.Debug {
				args = append(args, "-d")
			}
			if a.All {
				args = append(args, "-a")
			}
			if a.Yolo {
				args = append(args, "--yolo")
			}
			args = append(args, a.ExtraArgs...)

			out, err := runGemini(args, runOpts{Cwd: a.Cwd, TimeoutMs: a.TimeoutMs})
			if err != nil {
				writeResult(req.ID, toolCallResult{Content: []toolContent{{Type: "text", Text: err.Error()}}, IsError: true})
				return
			}
			writeResult(req.ID, toolCallResult{Content: []toolContent{{Type: "text", Text: out}}, IsError: false})
			return

		case "gemini_raw":
			var a struct {
				Args      []string `json:"args"`
				Cwd       string   `json:"cwd"`
				TimeoutMs int      `json:"timeoutMs"`
			}
			if err := json.Unmarshal(p.Arguments, &a); err != nil {
				writeError(req.ID, -32602, "invalid arguments", err.Error())
				return
			}
			if len(a.Args) == 0 {
				writeError(req.ID, -32602, "args array is required", nil)
				return
			}
			out, err := runGemini(a.Args, runOpts{Cwd: a.Cwd, TimeoutMs: a.TimeoutMs})
			if err != nil {
				writeResult(req.ID, toolCallResult{Content: []toolContent{{Type: "text", Text: err.Error()}}, IsError: true})
				return
			}
			writeResult(req.ID, toolCallResult{Content: []toolContent{{Type: "text", Text: out}}, IsError: false})
			return

		default:
			writeError(req.ID, -32602, "unknown tool: "+p.Name, nil)
			return
		}

	case "ping":
		writeResult(req.ID, map[string]any{"ok": true, "now": time.Now().UTC().Format(time.RFC3339)})
		return

	default:
		writeError(req.ID, -32601, "Method not found: "+req.Method, nil)
		return
	}
}

// ---------- main loop ----------
func main() {
	reader := bufio.NewReader(os.Stdin)
	sc := bufio.NewScanner(reader)
	buf := make([]byte, 0, 1024*1024)
	sc.Buffer(buf, 16*1024*1024)

	for {
		if !sc.Scan() {
			if err := sc.Err(); err != nil && !errors.Is(err, io.EOF) {
				fmt.Fprintf(os.Stderr, "[scan-error] %v\n", err)
			}
			break
		}
		line := sc.Text()
		if len(line) == 0 {
			continue
		}
		var req jsonrpcReq
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			fmt.Fprintf(os.Stderr, "[parse-error] %v\n", err)
			continue
		}
		handle(req)
	}
}
