package main

type ConfigType struct {
	BotToken string `toml:"bot_token"`
	AppID    int    `toml:"app_id"`
	AppHash  string `toml:"app_hash"`
	DBName   string `toml:"db_name"`
}

type UserInfo struct {
	ID       int64
	UserID   int64
	Username string
	State    string
}

type ConfInfo struct {
	ID        int64
	Title     string
	Post      string
	Banner    string
	ChannelID int64
	CatalogID int64
}
