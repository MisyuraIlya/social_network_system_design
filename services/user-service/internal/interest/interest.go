package interest

type City struct {
	ID   uint64 `gorm:"primaryKey" json:"id"`
	Name string `gorm:"uniqueIndex;size:120" json:"name"`
}
type Interest struct {
	ID   uint64 `gorm:"primaryKey" json:"id"`
	Name string `gorm:"uniqueIndex;size:120" json:"name"`
}
type InterestUser struct {
	UserID     string `gorm:"primaryKey;size:64"`
	InterestID uint64 `gorm:"primaryKey"`
}
