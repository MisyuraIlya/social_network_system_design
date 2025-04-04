package userdata

type UserData struct {
	UserID      int `gorm:"primary_key"`
	Description string
	CityID      int
	Education   string `gorm:"type:jsonb"`
	Hobby       string `gorm:"type:jsonb"`
}
