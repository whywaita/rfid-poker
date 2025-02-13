package config

var Conf Config

type Config struct {
	CardIDs map[string]string `required:"true" yaml:"card_ids"` // key: uid value: card

	HTTPMode bool `default:"true"`
}
