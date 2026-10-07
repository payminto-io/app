package service

import (
	"strconv"

	"github.com/payminto/payminto/backend/internal/ledger"
	"gorm.io/gorm"
)

func journalDB(j *ledger.Service) *gorm.DB { return j.DB() }

func idStr(id uint) string { return strconv.FormatUint(uint64(id), 10) }
