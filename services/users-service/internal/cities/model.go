package cities

type City struct {
	ID   int `gorm:"primary_key;auto_increment"`
	Name string
}
