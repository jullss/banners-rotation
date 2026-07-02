package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTP  HTTPConfig  `yaml:"http"`
	DB    DBConfig    `yaml:"db"`
	Kafka KafkaConfig `yaml:"kafka"`
}

type HTTPConfig struct {
	Addr string `yaml:"addr"`
}

type DBConfig struct {
	DSN string `yaml:"dsn"`
}

type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic"`
}

func New() (*Config, error) {
	path := "configs/config.yaml"
	if v := os.Getenv("CONFIG_PATH"); v != "" {
		path = v
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
