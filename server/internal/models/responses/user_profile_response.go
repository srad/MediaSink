package responses

import "time"

// UserProfileResponse is the client-facing projection of db.User. It
// deliberately omits the password hash.
type UserProfileResponse struct {
	UserID    uint      `json:"userId" extensions:"!x-nullable"`
	Username  string    `json:"username" extensions:"!x-nullable"`
	CreatedAt time.Time `json:"createdAt" extensions:"!x-nullable"`
	UpdatedAt time.Time `json:"updatedAt" extensions:"!x-nullable"`
}
