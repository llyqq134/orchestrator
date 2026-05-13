package config

type Config struct {
	Worker  WorkerServer  `yaml:"worker"`
	Manager ManagerServer `yaml:"manager"`
	DataDir string        `yaml:"data_dir"`
}
