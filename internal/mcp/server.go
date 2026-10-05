package mcp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"go-sd/internal/config"
	"go-sd/internal/generation"
	"go-sd/internal/kids"
	"go-sd/internal/llm"
	"go-sd/internal/preset"
	"go-sd/internal/sd"
	"go-sd/internal/settings"
)

type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     func(s *Server, args map[string]any) (string, error)
}

type Deps struct {
	DB       *preset.DB
	Gen      *generation.Service
	SD       sd.Service
	LLM      llm.Service
	Settings *settings.Service
	Kids     *kids.Manager
	Cfg      *config.Config
	OutDir   string
}

type Server struct {
	deps      Deps
	tools     map[string]Tool
	toolOrder []string
	mu        sync.RWMutex
	execMu    sync.Mutex
	writeMu   sync.Mutex
}

func NewServer(deps Deps) *Server {
	return &Server{deps: deps, tools: map[string]Tool{}}
}

func (s *Server) Register(t Tool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, dup := s.tools[t.Name]; dup {
		log.Fatalf("mcp: duplicate tool %q", t.Name)
	}
	s.tools[t.Name] = t
	s.toolOrder = append(s.toolOrder, t.Name)
}

func props(props map[string]any, required ...string) map[string]any {
	if props == nil {
		props = map[string]any{}
	}
	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func prop(desc, typ string, extra ...map[string]any) map[string]any {
	p := map[string]any{"type": typ, "description": desc}
	for _, e := range extra {
		for k, v := range e {
			p[k] = v
		}
	}
	return p
}

type NoopEmitter struct{}

func (NoopEmitter) Emit(event string, data ...any) {}

type NoopSessions struct{}

func (NoopSessions) AddToSession(imageBase64 string, info json.RawMessage, source string, isPreview bool, presetID *int64) int64 {
	return 0
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type callToolResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

const (
	protocolVersion   = "2024-11-05"
	errInvalidArgs    = -32602
	errMethodNotFound = -32601
	maxLineLen        = 64 << 20
)

var errLineTooLong = errors.New("line too long")

func (s *Server) Serve(r io.Reader, w io.Writer) error {
	br := bufio.NewReaderSize(r, 1<<20)
	out := bufio.NewWriter(w)
	var wg sync.WaitGroup
	for {
		raw, err := readLine(br, maxLineLen)
		if err == errLineTooLong {
			s.write(out, rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"),
				Error: &rpcError{Code: errInvalidArgs, Message: fmt.Sprintf("line exceeds %d bytes", maxLineLen)}})
			continue
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			wg.Wait()
			return err
		}
		line := strings.TrimSpace(string(raw))
		if line == "" {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			s.write(out, rpcResponse{JSONRPC: "2.0", ID: json.RawMessage("null"),
				Error: &rpcError{Code: -32700, Message: "parse error"}})
			continue
		}
		if len(req.ID) == 0 {
			continue
		}
		wg.Add(1)
		go func(req rpcRequest) {
			defer wg.Done()
			s.write(out, s.dispatch(req))
		}(req)
	}
	wg.Wait()
	return nil
}

func readLine(br *bufio.Reader, max int) ([]byte, error) {
	var buf []byte
	for {
		chunk, err := br.ReadSlice('\n')
		buf = append(buf, chunk...)
		if err == nil {
			return buf, nil
		}
		if err == bufio.ErrBufferFull {
			if len(buf) > max {
				for {
					_, e := br.ReadSlice('\n')
					if e == nil || e != bufio.ErrBufferFull {
						break
					}
				}
				return nil, errLineTooLong
			}
			continue
		}
		if err == io.EOF {
			if len(buf) > 0 {
				return buf, nil
			}
			return nil, io.EOF
		}
		return nil, err
	}
}

func (s *Server) write(out *bufio.Writer, resp rpcResponse) {
	b, err := json.Marshal(resp)
	if err != nil {
		return
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if _, err := out.Write(b); err != nil {
		return
	}
	_ = out.WriteByte('\n')
	out.Flush()
}

func (s *Server) dispatch(req rpcRequest) rpcResponse {
	switch req.Method {
	case "initialize":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "sd-mcp", "version": "0.1.0"},
		}}
	case "ping":
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case "tools/list":
		s.mu.RLock()
		defer s.mu.RUnlock()
		tools := make([]map[string]any, 0, len(s.toolOrder))
		for _, name := range s.toolOrder {
			t := s.tools[name]
			tools = append(tools, map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"inputSchema": t.InputSchema,
			})
		}
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": tools}}
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments,omitempty"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil || p.Name == "" {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID,
				Error: &rpcError{Code: errInvalidArgs, Message: "bad tools/call params"}}
		}
		s.mu.RLock()
		t, ok := s.tools[p.Name]
		s.mu.RUnlock()
		if !ok {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID,
				Error: &rpcError{Code: errMethodNotFound, Message: fmt.Sprintf("unknown tool %q", p.Name)}}
		}
		args := map[string]any{}
		if len(p.Arguments) > 0 {
			if err := json.Unmarshal(p.Arguments, &args); err != nil {
				return rpcResponse{JSONRPC: "2.0", ID: req.ID,
					Error: &rpcError{Code: errInvalidArgs, Message: "arguments must be an object"}}
			}
		}
		text, err := s.execTool(t, args)
		if err != nil {
			return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: callToolResult{
				Content: []toolContent{{Type: "text", Text: err.Error()}}, IsError: true}}
		}
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: callToolResult{
			Content: []toolContent{{Type: "text", Text: text}}}}
	}
	return rpcResponse{JSONRPC: "2.0", ID: req.ID,
		Error: &rpcError{Code: errMethodNotFound, Message: "method not found: " + req.Method}}
}

func (s *Server) execTool(t Tool, args map[string]any) (string, error) {
	if t.Name == "interrupt" {
		return t.Handler(s, args)
	}
	s.execMu.Lock()
	defer s.execMu.Unlock()
	return t.Handler(s, args)
}

func argString(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func argFloat(args map[string]any, key string) float64 {
	switch v := args[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	}
	return 0
}

func argInt(args map[string]any, key string) int64 {
	return int64(argFloat(args, key))
}

func argBool(args map[string]any, key string) bool {
	v, _ := args[key].(bool)
	return v
}

func argStringSlice(args map[string]any, key string) []string {
	raw, _ := args[key].([]any)
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if str, ok := v.(string); ok {
			out = append(out, str)
		}
	}
	return out
}

func optString(args map[string]any, key string) *string {
	if _, ok := args[key]; !ok {
		return nil
	}
	v := argString(args, key)
	return &v
}

func optInt(args map[string]any, key string) *int64 {
	if _, ok := args[key]; !ok {
		return nil
	}
	v := argInt(args, key)
	return &v
}

func optBool(args map[string]any, key string, def bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return def
}

func toJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(b)
}
