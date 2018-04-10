package auction_butler

type DatabaseConfig struct {
	Driver string `json:"driver"`
	Source string `json:"source"`
}

type Config struct {
	Debug                    bool           `json:"debug"`
	Token                    string         `json:"token"`
	ChatID                   int64          `json:"chat_id"`
	Database                 DatabaseConfig `json:"database"`
	ReminderAnnounceInterval Duration       `json:"reminder_announce_interval"`
	CountdownFrom            int64          `json:"countdown_from"`
	ResettingCountdownFrom   int64          `json:"resetting_countdown_from"`
	MsgDeleteCounter         Duration       `json:"msg_destroy_counter"`
	ConversionFactor         int64          `json:"conversion_factor"`
}
