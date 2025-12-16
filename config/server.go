package config

type Server struct {
	Host string `yaml:"HOST" env:"HOST" env-default:"localhost"`
	Port string `yaml:"PORT" env:"PORT" env-default:"8080"`
}
