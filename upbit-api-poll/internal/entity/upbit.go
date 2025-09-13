package entity

import "time"

type NewsTitle = string

//easyjson:json
type SingleAnnouncement struct {
	Success      bool   `json:"success"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Data         Notice `json:"data"`
}

func (a SingleAnnouncement) NewsTitle() NewsTitle {
	return a.Data.Title
}

//easyjson:json
type Announcements struct {
	Success      bool   `json:"success"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Data         struct {
		TotalPages int      `json:"total_pages"`
		TotalCount int      `json:"total_count"`
		Notices    []Notice `json:"notices"`
	} `json:"data"`
}

type Notice struct {
	ID            int       `json:"id"`
	Category      string    `json:"category"`
	Title         string    `json:"title"`
	ListedAt      time.Time `json:"listed_at"`
	FirstListedAt time.Time `json:"first_listed_at"`
}
