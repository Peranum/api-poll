package entity

//easyjson:json
type Detection struct {
	Ticker string `json:"ticker"`
}

//easyjson:json
type SuckerPayload struct {
	ID            int64       `json:"id"`
	Time          int64       `json:"time"`
	Announcement  string      `json:"announcement"`
	OriginalTitle string      `json:"original_title"`
	URL           string      `json:"url"`
	Exchange      string      `json:"exchange"`
	Type          string      `json:"type"`
	Detections    []Detection `json:"detections"`
}
