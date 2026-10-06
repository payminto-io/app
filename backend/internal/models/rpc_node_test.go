package models

import "testing"

func TestRPCNode_TableName(t *testing.T) {
	if (RPCNode{}).TableName() != "rpc_nodes" {
		t.Error("wrong table name")
	}
}

func TestRPCNodeStatusConstants(t *testing.T) {
	if RPCNodeStatusHealthy != "healthy" {
		t.Error("wrong status constant")
	}
}
