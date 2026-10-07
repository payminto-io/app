package cre

import (
	"encoding/json"
	"fmt"
)

// MetadataLength is the forwarder's report header: workflow id (32), workflow name (10), owner (20), report id (2).
const MetadataLength = 64

// Metadata is the decoded forwarder header.
type Metadata struct {
	WorkflowID   [32]byte
	WorkflowName [10]byte
	Owner        [20]byte
	ReportID     [2]byte
}

func DecodeMetadata(b []byte) (Metadata, error) {
	if len(b) != MetadataLength {
		return Metadata{}, fmt.Errorf("%w: metadata is %d bytes, want %d", ErrInvalidReport, len(b), MetadataLength)
	}
	var m Metadata
	copy(m.WorkflowID[:], b[0:32])
	copy(m.WorkflowName[:], b[32:42])
	copy(m.Owner[:], b[42:62])
	copy(m.ReportID[:], b[62:64])
	return m, nil
}

func (m Metadata) Encode() []byte {
	out := make([]byte, 0, MetadataLength)
	out = append(out, m.WorkflowID[:]...)
	out = append(out, m.WorkflowName[:]...)
	out = append(out, m.Owner[:]...)
	out = append(out, m.ReportID[:]...)
	return out
}

func jsonUnmarshal(s string, v any) error { return json.Unmarshal([]byte(s), v) }
