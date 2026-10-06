package models

// ActivityLog is the request audit trail. Captured by the activity-log
// middleware on every API request. Uses BaseModel because it's an
// audit-style table.
type ActivityLog struct {
	BaseModel
	MemberID            *uint   `gorm:"index" json:"memberID,omitempty"`
	ExternalPlatformID  *uint   `gorm:"index" json:"externalPlatformID,omitempty"`
	SessionID           *string `gorm:"type:varchar(128);index" json:"sessionID,omitempty"`
	Method              string  `gorm:"type:varchar(10);not null" json:"method"`
	Path                string  `gorm:"type:text;not null" json:"path"`
	StatusCode          int     `gorm:"not null;index" json:"statusCode"`
	DurationMs          int64   `json:"durationMs"`
	IPAddress           *string `gorm:"type:varchar(45)" json:"ipAddress,omitempty"`
	UserAgent           *string `gorm:"type:text" json:"userAgent,omitempty"`
	RequestBodyPreview  *string `gorm:"type:text" json:"requestBodyPreview,omitempty"`
	ResponseBodyPreview *string `gorm:"type:text" json:"responseBodyPreview,omitempty"`
	EventName           *string `gorm:"type:varchar(100);index" json:"eventName,omitempty"`
	EventCategory       *string `gorm:"type:varchar(50);index" json:"eventCategory,omitempty"`
	Action              *string `gorm:"type:varchar(50)" json:"action,omitempty"`
	GeoCountry          *string `gorm:"type:varchar(2)" json:"geoCountry,omitempty"`
	GeoCity             *string `gorm:"type:varchar(100)" json:"geoCity,omitempty"`
	ErrorMessage        *string `gorm:"type:text" json:"errorMessage,omitempty"`
}

func (ActivityLog) TableName() string { return "activity_logs" }
