package profile

import "time"

type Profile struct {
	UserID      string         `gorm:"primaryKey;size:64" json:"user_id"`
	Description string         `json:"description"`
	CityID      uint64         `json:"city_id"`
	Education   map[string]any `gorm:"type:jsonb" json:"education"`
	Hobby       map[string]any `gorm:"type:jsonb" json:"hobby"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type UpsertReq struct {
	Description string         `json:"description"`
	CityID      uint64         `json:"city_id"`
	Education   map[string]any `json:"education"`
	Hobby       map[string]any `json:"hobby"`
}
