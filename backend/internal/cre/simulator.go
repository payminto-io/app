package cre

import "bytes"

// The CRE simulator (cre workflow simulate) signs every report with one fixed identity: workflow id 0x11..11,
// owner 0xaa..aa, name HashTruncateName(workflow name). A record carrying it is simulated, whatever the provider.
var (
	SimulatorWorkflowID = [32]byte{0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11, 0x11}
	SimulatorOwner      = [20]byte{0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa, 0xaa}
)

// IsSimulatorIdentity reports whether metadata carries the simulator's fixed workflow id or owner.
func IsSimulatorIdentity(m Metadata) bool {
	return bytes.Equal(m.WorkflowID[:], SimulatorWorkflowID[:]) || bytes.Equal(m.Owner[:], SimulatorOwner[:])
}

// IsSimulatorBinding reports whether a configured binding is the simulator's identity.
func IsSimulatorBinding(b Binding) bool {
	return bytes.Equal(b.ID[:], SimulatorWorkflowID[:]) || bytes.Equal(b.Owner[:], SimulatorOwner[:])
}
