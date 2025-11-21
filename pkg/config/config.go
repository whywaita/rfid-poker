package config

var Conf Config

type Config struct {
	CardIDs map[string]string `required:"true" yaml:"card_ids"` // key: uid value: card

	// GameTimeoutSeconds is the number of seconds to wait for card reads before automatically ending the game
	// If set to 0, timeout is disabled. Default: 10
	GameTimeoutSeconds int `env:"RFID_POKER_CLIENT_TIMEOUT_SECONDS" default:"10"`

	MySQLUser     string `required:"true" env:"RFID_POKER_MYSQL_USER"`
	MySQLPass     string `required:"true" env:"RFID_POKER_MYSQL_PASS"`
	MySQLHost     string `required:"true" env:"RFID_POKER_MYSQL_HOST"`
	MySQLPort     string `required:"true" env:"RFID_POKER_MYSQL_PORT"`
	MySQLDatabase string `required:"true" env:"RFID_POKER_MYSQL_DATABASE"`
}
