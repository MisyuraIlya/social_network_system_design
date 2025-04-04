package userdata

type UpdateUserDataRequest struct {
	Description string `json:"description"`
	CityID      int    `json:"city_id"`
	Education   string `json:"education"`
	Hobby       string `json:"hobby"`
}

type GetUserDataResponse struct {
	UserID      int    `json:"user_id"`
	Description string `json:"description"`
	CityID      int    `json:"city_id"`
	Education   string `json:"education"`
	Hobby       string `json:"hobby"`
}
