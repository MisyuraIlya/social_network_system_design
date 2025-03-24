package user

type UserCreatePayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type UserUpdatePayload struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
