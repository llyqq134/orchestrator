package config

type Config struct {
	Worker WorkerServer `yaml:"worker"`
	Manager ManagerServer `yaml:"manager"`
}
