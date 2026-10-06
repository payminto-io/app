package models

// AnalyticsGroup is a named query group like "Daily Volume", "Top Customers",
// or "Revenue by Chain". Groups belong to a category (overview, payments,
// withdrawals, sweeps, custom). The dashboard renders one tile per group.
type AnalyticsGroup struct {
	PaymintoModel
	Name        string  `gorm:"type:varchar(200);not null" json:"name"`
	Slug        string  `gorm:"type:varchar(200);not null;uniqueIndex" json:"slug"`
	Category    string  `gorm:"type:varchar(50);not null;index" json:"category"`
	Description *string `gorm:"type:text" json:"description,omitempty"`
	IsDefault   bool    `gorm:"default:false" json:"isDefault"`
	OrderIndex  int     `gorm:"default:0" json:"orderIndex"`

	Graphs  []AnalyticsGraph       `gorm:"foreignKey:GroupID" json:"graphs,omitempty"`
	Filters []AnalyticsGroupFilter `gorm:"foreignKey:GroupID" json:"filters,omitempty"`
}

func (AnalyticsGroup) TableName() string { return "analytics_groups" }

// AnalyticsGraph holds the chart configuration for a group — chart type,
// X axis, Y axis, group-by clause. The actual data is computed at query
// time by AnalyticsService.FetchData.
type AnalyticsGraph struct {
	PaymintoModel
	GroupID     uint    `gorm:"not null;index" json:"groupID"`
	Name        string  `gorm:"type:varchar(200);not null" json:"name"`
	ChartType   string  `gorm:"type:varchar(20);not null" json:"chartType"`
	XAxisField  string  `gorm:"type:varchar(100)" json:"xAxisField"`
	YAxisField  string  `gorm:"type:varchar(100)" json:"yAxisField"`
	GroupBy     *string `gorm:"type:varchar(100)" json:"groupBy,omitempty"`
	Aggregation string  `gorm:"type:varchar(20);default:'sum'" json:"aggregation"`
	Query       *string `gorm:"type:text" json:"query,omitempty"`
	OrderIndex  int     `gorm:"default:0" json:"orderIndex"`

	Group *AnalyticsGroup `gorm:"foreignKey:GroupID" json:"-"`
}

func (AnalyticsGraph) TableName() string { return "analytics_graphs" }

// AnalyticsFilter is a reusable filter definition (e.g., "last 7 days",
// "currency = USDT", "blockchain = ETH"). Filters can be attached to groups.
type AnalyticsFilter struct {
	PaymintoModel
	Name       string `gorm:"type:varchar(200);not null" json:"name"`
	Field      string `gorm:"type:varchar(100);not null" json:"field"`
	Operator   string `gorm:"type:varchar(20);not null" json:"operator"`
	Value      string `gorm:"type:text" json:"value"`
	FilterType string `gorm:"type:varchar(20);default:'static'" json:"filterType"`
}

func (AnalyticsFilter) TableName() string { return "analytics_filters" }

// AnalyticsCustomFilter is a user-defined filter saved against a member's
// dashboard. Used for things like "show only my Sepolia payments".
type AnalyticsCustomFilter struct {
	PaymintoModel
	MemberID    uint    `gorm:"not null;index" json:"memberID"`
	Name        string  `gorm:"type:varchar(200);not null" json:"name"`
	Description *string `gorm:"type:text" json:"description,omitempty"`
	Definition  string  `gorm:"type:text;not null" json:"definition"`
	IsShared    bool    `gorm:"default:false" json:"isShared"`

	Member *Member `gorm:"foreignKey:MemberID" json:"-"`
}

func (AnalyticsCustomFilter) TableName() string { return "analytics_custom_filters" }

// AnalyticsGroupFilter joins groups to filters with display order.
type AnalyticsGroupFilter struct {
	PaymintoModel
	GroupID    uint `gorm:"not null;index" json:"groupID"`
	FilterID   uint `gorm:"not null;index" json:"filterID"`
	OrderIndex int  `gorm:"default:0" json:"orderIndex"`
	Required   bool `gorm:"default:false" json:"required"`

	Group  *AnalyticsGroup  `gorm:"foreignKey:GroupID" json:"-"`
	Filter *AnalyticsFilter `gorm:"foreignKey:FilterID" json:"-"`
}

func (AnalyticsGroupFilter) TableName() string { return "analytics_group_filters" }

// AnalyticsUserGroup grants a Member access to an AnalyticsGroup. PayRam uses
// this to scope which dashboards a user can see.
type AnalyticsUserGroup struct {
	PaymintoModel
	MemberID uint   `gorm:"not null;index" json:"memberID"`
	GroupID  uint   `gorm:"not null;index" json:"groupID"`
	Role     string `gorm:"type:varchar(20);default:'viewer'" json:"role"`

	Member *Member         `gorm:"foreignKey:MemberID" json:"-"`
	Group  *AnalyticsGroup `gorm:"foreignKey:GroupID" json:"-"`
}

func (AnalyticsUserGroup) TableName() string { return "analytics_user_groups" }

// Analytics category constants.
const (
	AnalyticsCategoryOverview    = "overview"
	AnalyticsCategoryPayments    = "payments"
	AnalyticsCategoryWithdrawals = "withdrawals"
	AnalyticsCategorySweeps      = "sweeps"
	AnalyticsCategoryReferrals   = "referrals"
	AnalyticsCategoryCustom      = "custom"
)

// Analytics chart types.
const (
	AnalyticsChartTypeLine  = "line"
	AnalyticsChartTypeBar   = "bar"
	AnalyticsChartTypePie   = "pie"
	AnalyticsChartTypeTable = "table"
	AnalyticsChartTypeStat  = "stat"
)
