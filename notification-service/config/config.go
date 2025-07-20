package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Port  string `yaml:"port" env:"PORT" env-default:"8084"`
	Kafka Kafka  `yaml:"kafka"`
	Email Email  `yaml:"email"`
}

type Kafka struct {
	Brokers           []string `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092" env-separator:","`
	NotificationTopic string   `yaml:"notification_topic" env:"KAFKA_NOTIFICATION_TOPIC" env-default:"notification-requests"`
	ConsumerGroup     string   `yaml:"consumer_group" env:"KAFKA_CONSUMER_GROUP" env-default:"notification-service"`
}

type Email struct {
	SMTPHost     string `yaml:"smtp_host" env:"SMTP_HOST" env-default:"smtp.gmail.com"`
	SMTPPort     int    `yaml:"smtp_port" env:"SMTP_PORT" env-default:"587"`
	SMTPUser     string `yaml:"smtp_user" env:"SMTP_USER" env-default:""`
	SMTPPassword string `yaml:"smtp_password" env:"SMTP_PASSWORD" env-default:""`
	FromEmail    string `yaml:"from_email" env:"FROM_EMAIL" env-default:"noreply@eventbooking.com"`
	FromName     string `yaml:"from_name" env:"FROM_NAME" env-default:"Event Booking System"`
}

func Initialise(configPath string, useEnv bool) (*Config, error) {
	cfg := &Config{}

	if useEnv {
		if err := cleanenv.ReadEnv(cfg); err != nil {
			return nil, fmt.Errorf("failed to read environment variables: %w", err)
		}
		return cfg, nil
	}

	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
				return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
			}
			return cfg, nil
		}
	}

	// Fallback to environment variables
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, fmt.Errorf("failed to read environment variables: %w", err)
	}

	return cfg, nil
}
