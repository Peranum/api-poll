package config

type UpbitAPI struct {
	AnnouncementsEndpoint          string  `mapstructure:"announcements_endpoint"               validate:"required" env:"UPBIT_API_ANNOUNCEMENTS_ENDPOINT"`
	AnnouncementsSingleIPMaxRPS    float64 `mapstructure:"announcements_single_ip_max_rps"      validate:"required" env:"UPBIT_API_ANNOUNCEMENTS_SINGLE_IP_MAX_RPS"`
	AnnouncementByIDEndpoint       string  `mapstructure:"announcement_by_id_endpoint"          validate:"required" env:"UPBIT_API_ANNOUNCEMENT_BY_ID_ENDPOINT"`
	AnnouncementByIDSingleIPMaxRPS float64 `mapstructure:"announcement_by_id_single_ip_max_rps" validate:"required" env:"UPBIT_API_ANNOUNCEMENT_BY_ID_SINGLE_IP_MAX_RPS"`
	NoticeByIDEndpoint             string  `mapstructure:"notice_by_id_endpoint"                validate:"required" env:"UPBIT_API_NOTICE_BY_ID_ENDPOINT"`
	NoticeByIDSingleIPMaxRPS       float64 `mapstructure:"notice_by_id_single_ip_max_rps"       validate:"required" env:"UPBIT_API_NOTICE_BY_ID_SINGLE_IP_MAX_RPS"`
}
