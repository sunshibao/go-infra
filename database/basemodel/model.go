package basemodel

import (
	"time"
)

type Model struct {
	ID        uint64 `gorm:"primary_key"`
	CreatedBy uint64
	CreatedAt time.Time `sql:"default:CURRENT_TIMESTAMP"`
	UpdatedBy uint64
	UpdatedAt time.Time `sql:"default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"`
}
