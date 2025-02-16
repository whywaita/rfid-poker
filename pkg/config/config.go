package config

var Conf Config

type Config struct {
	CardIDs map[string]string `required:"true" yaml:"card_ids"` // key: uid value: card

	MySQLUser     string `required:"true" env:"RFID_POKER_MYSQL_USER"`
	MySQLPass     string `required:"true" env:"RFID_POKER_MYSQL_PASS"`
	MySQLHost     string `required:"true" env:"RFID_POKER_MYSQL_HOST"`
	MySQLPort     string `required:"true" env:"RFID_POKER_MYSQL_PORT"`
	MySQLDatabase string `required:"true" env:"RFID_POKER_MYSQL_DATABASE"`
}
