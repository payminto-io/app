package models

import "time"

// RPCNode is one RPC endpoint for a blockchain. A blockchain has many
// RPCNodes; the pool picks a healthy one per request and fails over when a
// node returns errors or times out.
// PayRam columns: blockchain_id, url, node_type, is_preferred, is_active,
// max_daily_calls, api_key, auth_username, auth_password, chain_identifier,
// credential_hash
type RPCNode struct {
	PaymintoModel
	BlockchainID    uint       `gorm:"not null;index" json:"blockchainID"`
	Name            string     `gorm:"type:varchar(100)" json:"name"`
	URL             string     `gorm:"type:text;not null" json:"url"`
	NodeType        string     `gorm:"type:varchar(20);default:'free'" json:"nodeType"`    // PayRam: free, paid
	IsPreferred     bool       `gorm:"default:false" json:"isPreferred"`                   // PayRam: preferred node for the chain
	IsActive        bool       `gorm:"default:true" json:"isActive"`                       // PayRam: active/inactive toggle
	MaxDailyCalls   *int       `json:"maxDailyCalls,omitempty"`                            // PayRam: rate limit per day
	APIKey          *string    `gorm:"type:text" json:"-"`                                 // PayRam: API key for paid nodes
	AuthUsername    *string    `gorm:"type:varchar(100)" json:"-"`                         // PayRam: basic auth username
	AuthPassword    *string    `gorm:"type:varchar(255)" json:"-"`                         // PayRam: basic auth password
	ChainIdentifier *string    `gorm:"type:varchar(100)" json:"chainIdentifier,omitempty"` // PayRam: chain ID or genesis hash for verification
	CredentialHash  *string    `gorm:"type:varchar(100)" json:"-"`                         // PayRam: hash of credentials for change detection
	AuthHeader      *string    `gorm:"type:text" json:"-"`                                 // Bearer token — never returned in JSON
	Weight          int        `gorm:"default:1;not null" json:"weight"`
	Priority        int        `gorm:"default:100;not null" json:"priority"`
	Status          string     `gorm:"type:varchar(20);default:'healthy';not null" json:"status"`
	LastHealthCheck *time.Time `json:"lastHealthCheck,omitempty"`
	FailCount       int        `gorm:"default:0" json:"failCount"`
	LastErrorAt     *time.Time `json:"lastErrorAt,omitempty"`
	LastError       *string    `gorm:"type:text" json:"lastError,omitempty"`

	Blockchain *Blockchain `gorm:"foreignKey:BlockchainID" json:"blockchain,omitempty"`
}

func (RPCNode) TableName() string { return "rpc_nodes" }

// Status constants.
const (
	RPCNodeStatusHealthy     = "healthy"
	RPCNodeStatusDegraded    = "degraded"
	RPCNodeStatusUnhealthy   = "unhealthy"
	RPCNodeStatusMaintenance = "maintenance"
)
