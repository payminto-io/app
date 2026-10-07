package cre

import (
	"crypto/sha256"
	"encoding/hex"
)

// KeystoneName is Keystone's HashTruncateName: the ASCII bytes of the first ten hex characters of
// sha256(name) (contracts/src/cre/WorkflowName.sol). Docs vector: "my_workflow" -> 0x62373666336165316465.
func KeystoneName(name string) [10]byte {
	digest := sha256.Sum256([]byte(name))
	var out [10]byte
	copy(out[:], hex.EncodeToString(digest[:5]))
	return out
}
