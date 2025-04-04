package interests

type Interest struct {
	ID   int `gorm:"primary_key;auto_increment"`
	Name string
}

type InterestUser struct {
	InterestID int
	UserID     int
}
