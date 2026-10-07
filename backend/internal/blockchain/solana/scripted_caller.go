package solana

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// ScriptedCaller answers RPC calls from registered handlers; tests in this and other packages
// drive the adapter, watcher and sweeper against recorded fixtures with it.
type ScriptedCaller struct {
	mu       sync.Mutex
	handlers map[string]func(params []any) (any, error)
	Calls    []ScriptedCall
}

// ScriptedCall records one invocation.
type ScriptedCall struct {
	Method string
	Params []any
}

// NewScriptedCaller returns an empty script; unknown methods fail loudly.
func NewScriptedCaller() *ScriptedCaller {
	return &ScriptedCaller{handlers: map[string]func(params []any) (any, error){}}
}

// On registers the handler for method; the returned value is JSON-encoded as the RPC result.
func (s *ScriptedCaller) On(method string, fn func(params []any) (any, error)) *ScriptedCaller {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = fn
	return s
}

// Result registers a constant result.
func (s *ScriptedCaller) Result(method string, result any) *ScriptedCaller {
	return s.On(method, func([]any) (any, error) { return result, nil })
}

// Count returns how many times method was called.
func (s *ScriptedCaller) Count(method string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, c := range s.Calls {
		if c.Method == method {
			n++
		}
	}
	return n
}

func (s *ScriptedCaller) Call(_ context.Context, method string, params []any, out any) error {
	s.mu.Lock()
	s.Calls = append(s.Calls, ScriptedCall{Method: method, Params: params})
	fn, ok := s.handlers[method]
	s.mu.Unlock()
	if !ok {
		return fmt.Errorf("scripted caller: no handler for %s", method)
	}
	res, err := fn(params)
	if err != nil {
		return err
	}
	if out == nil || res == nil {
		return nil
	}
	raw, err := json.Marshal(res)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

// ContextValue wraps a value the way RPC methods with context do.
func ContextValue(slot uint64, value any) map[string]any {
	return map[string]any{"context": map[string]any{"slot": slot}, "value": value}
}

// LoadFixtureJSON reads a testdata file as raw JSON.
func LoadFixtureJSON(path string) (json.RawMessage, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

// LoadFixtureTransaction decodes a jsonParsed transaction fixture.
func LoadFixtureTransaction(path string) (*ParsedTransaction, error) {
	raw, err := LoadFixtureJSON(path)
	if err != nil {
		return nil, err
	}
	var tx ParsedTransaction
	if err := json.Unmarshal(raw, &tx); err != nil {
		return nil, err
	}
	return &tx, nil
}

// FirstParamString returns params[0] as a string when it is one.
func FirstParamString(params []any) string {
	if len(params) == 0 {
		return ""
	}
	s, _ := params[0].(string)
	return s
}

// Handler returns the registered handler for method so a test can wrap it.
func (s *ScriptedCaller) Handler(method string) func(params []any) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.handlers[method]
}
