package service

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"

	"github.com/payminto/payminto/backend/internal/blockchain/solana"
)

// aliasCaller lets sweep tests name signatures ("SIG1") while the sweeper persists the real one it
// signed: the fake a scripted sendTransaction returns becomes an alias of the real signature, params
// carry the fake to the handlers and results carry the real one back.
type aliasCaller struct {
	inner  *solana.ScriptedCaller
	mu     sync.Mutex
	toFake map[string]string
	toReal map[string]string
}

func newAliasCaller(inner *solana.ScriptedCaller) *aliasCaller {
	return &aliasCaller{inner: inner, toFake: map[string]string{}, toReal: map[string]string{}}
}

// real returns the signature the sweeper signed for a scripted fake name (the fake itself when unknown).
func (a *aliasCaller) real(fake string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if r, ok := a.toReal[fake]; ok {
		return r
	}
	return fake
}

func (a *aliasCaller) params(method string, params []any) []any {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]any, len(params))
	for i, p := range params {
		switch v := p.(type) {
		case string:
			if f, ok := a.toFake[v]; ok {
				p = f
			}
		case []string:
			mapped := make([]string, len(v))
			for j, s := range v {
				mapped[j] = s
				if f, ok := a.toFake[s]; ok {
					mapped[j] = f
				}
			}
			p = mapped
		}
		out[i] = p
	}
	return out
}

func (a *aliasCaller) result(raw json.RawMessage, out any) error {
	if len(raw) == 0 || out == nil {
		return nil
	}
	a.mu.Lock()
	for fake, real := range a.toReal {
		raw = bytes.ReplaceAll(raw, []byte(`"`+fake+`"`), []byte(`"`+real+`"`))
	}
	a.mu.Unlock()
	return json.Unmarshal(raw, out)
}

func (a *aliasCaller) sent(method string, params []any, raw json.RawMessage) {
	if method != "sendTransaction" || len(raw) == 0 {
		return
	}
	tx, err := solana.DecodeTransactionBase64(solana.FirstParamString(params))
	if err != nil {
		return
	}
	var fake string
	if json.Unmarshal(raw, &fake) != nil || fake == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.toFake[tx.Signature()] = fake
	a.toReal[fake] = tx.Signature()
}

func (a *aliasCaller) Call(ctx context.Context, method string, params []any, out any) error {
	var raw json.RawMessage
	if err := a.inner.Call(ctx, method, a.params(method, params), &raw); err != nil {
		return err
	}
	a.sent(method, params, raw)
	return a.result(raw, out)
}

func (a *aliasCaller) NodeIDs() []uint { return a.inner.NodeIDs() }

func (a *aliasCaller) CallOn(ctx context.Context, nodeID uint, method string, params []any, out any) error {
	var raw json.RawMessage
	if err := a.inner.CallOn(ctx, nodeID, method, a.params(method, params), &raw); err != nil {
		return err
	}
	a.sent(method, params, raw)
	return a.result(raw, out)
}
