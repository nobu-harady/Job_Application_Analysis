package models

import (
	"time"

	"gorm.io/gorm"
)

// 顧客情報を表すモデル
type Customer struct {
	gorm.Model                      // ID, CreatedAt, UpdatedAt, DeletedAt を自動的に含める
	RecruitmentMethod     string    `gorm:"not null"`
	CustomerName          string    `gorm:"not null"`
	YearMonth             time.Time `gorm:"not null"`
	MonthlyFee            int
	MonthlyApplications   int
	MonthlyRegistrations  int
	MonthlyPlacements     int
	PlacementUnitPrice    float64
	RegistrationUnitPrice float64
	ApplicationUnitPrice  float64
}
