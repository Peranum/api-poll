package entity

import (
	"time"

	"github.com/google/uuid"
)

type NewsTitle = string

type News struct {
	ID                uuid.UUID
	Title             string
	ListedAt          time.Time
	FirstListedAt     time.Time
	ReceivedFromAPIAt time.Time
}

//easyjson:json
type Announcements struct {
	Success bool `json:"success"`
	Data    struct {
		TotalPages int `json:"total_pages"`
		TotalCount int `json:"total_count"`
		Notices    []struct {
			ListedAt        time.Time `json:"listed_at"`
			FirstListedAt   time.Time `json:"first_listed_at"`
			ID              int       `json:"id"`
			Title           string    `json:"title"`
			Category        string    `json:"category"`
			NeedNewBadge    bool      `json:"need_new_badge"`
			NeedUpdateBadge bool      `json:"need_update_badge"`
		} `json:"notices"`
		FixedNotices []interface{} `json:"fixed_notices"`
	} `json:"data"`
}
