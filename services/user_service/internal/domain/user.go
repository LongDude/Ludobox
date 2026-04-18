package domain

type User struct {
	ID         int      `json:"user_id"`
	FirstName  string   `json:"first_name"`
	LastName   string   `json:"last_name"`
	Email      string   `json:"email"`
	Roles      []string `json:"roles"`
	LocaleType *string  `json:"locale_type"`
	NickName   string   `json:"nickname"`
	Balance    int      `json:"balance"`
}
