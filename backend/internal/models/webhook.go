package models

import "time"

type Webhook struct {
	PaymintoModel
	URL                string `gorm:"type:text;not null" json:"url"`
	Secret             string `gorm:"type:text;not null" json:"-"`
	Events             string `gorm:"type:text" json:"events"`
	Active             bool   `gorm:"default:true" json:"active"`
	ExternalPlatformID uint   `gorm:"not null" json:"externalPlatformID"`

	ExternalPlatform *ExternalPlatform    `gorm:"foreignKey:ExternalPlatformID" json:"-"`
	DeliveryLogs     []WebhookDeliveryLog `gorm:"foreignKey:WebhookID" json:"-"`
}

func (Webhook) TableName() string { return "webhooks" }

type WebhookDeliveryLog struct {
	PaymintoModel
	Event        string     `gorm:"type:varchar(50);not null" json:"event"`
	Payload      string     `gorm:"type:text;not null" json:"payload"`
	Status       string     `gorm:"type:varchar(20);default:'pending'" json:"status"`
	ResponseCode *int       `json:"responseCode,omitempty"`
	ResponseBody *string    `gorm:"type:text" json:"responseBody,omitempty"`
	Attempts     int        `gorm:"default:0" json:"attempts"`
	NextRetryAt  *time.Time `json:"nextRetryAt,omitempty"`
	DeliveredAt  *time.Time `json:"deliveredAt,omitempty"`

	WebhookID        uint  `gorm:"not null" json:"webhookID"`
	PaymentRequestID *uint `json:"paymentRequestID,omitempty"`

	Webhook *Webhook `gorm:"foreignKey:WebhookID" json:"-"`
}

func (WebhookDeliveryLog) TableName() string { return "webhook_delivery_logs" }
