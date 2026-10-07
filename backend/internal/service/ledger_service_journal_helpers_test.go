package service

import (
	"github.com/payminto/payminto/backend/internal/ledger"
	"gorm.io/gorm"
)

func journalDB(j *ledger.Service) *gorm.DB { return j.DB() }
