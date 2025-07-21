package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Port         string       `yaml:"port" env:"PORT" env-default:"8083"`
	JWTSecret    string       `yaml:"jwt_secret" env:"JWT_SECRET" env-required:"true"`
	Database     Database     `yaml:"database"`
	Redis        Redis        `yaml:"redis"`
	Kafka        Kafka        `yaml:"kafka"`
	EventService EventService `yaml:"event_service"`
	Worker       Worker       `yaml:"worker"`
}

type Worker struct {
	MaxWorkers int `yaml:"max_workers" env:"WORKER_MAX_WORKERS" env-default:"20"`
}

type Database struct {
	User         string `yaml:"user" env:"DB_USER" env-required:"true"`
	Password     string `yaml:"password" env:"DB_PASSWORD" env-required:"true"`
	DatabaseName string `yaml:"database_name" env:"DB_NAME" env-required:"true"`
	Host         string `yaml:"host" env:"DB_HOST" env-default:"localhost"`
	Port         string `yaml:"port" env:"DB_PORT" env-default:"5432"`
	SSLMode      string `yaml:"ssl_mode" env:"DB_SSL_MODE" env-default:"disable"`

	// Connection Pool Settings
	MaxOpenConns    int `yaml:"max_open_conns" env:"DB_MAX_OPEN_CONNS" env-default:"25"`
	MaxIdleConns    int `yaml:"max_idle_conns" env:"DB_MAX_IDLE_CONNS" env-default:"10"`
	ConnMaxLifetime int `yaml:"conn_max_lifetime_minutes" env:"DB_CONN_MAX_LIFETIME" env-default:"30"`
}

func (d *Database) GetDatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.DatabaseName, d.SSLMode)
}

type Redis struct {
	Host     string `yaml:"host" env:"REDIS_HOST" env-default:"localhost"`
	Port     string `yaml:"port" env:"REDIS_PORT" env-default:"6379"`
	Password string `yaml:"password" env:"REDIS_PASSWORD" env-default:""`
	DB       int    `yaml:"db" env:"REDIS_DB" env-default:"0"`
}

func (r *Redis) GetRedisURL() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

type Kafka struct {
	Brokers           []string `yaml:"brokers" env:"KAFKA_BROKERS" env-default:"localhost:9092" env-separator:","`
	BookingTopic      string   `yaml:"booking_topic" env:"KAFKA_BOOKING_TOPIC" env-default:"booking-requests"`
	NotificationTopic string   `yaml:"notification_topic" env:"KAFKA_NOTIFICATION_TOPIC" env-default:"notification-requests"`
	ConsumerGroup     string   `yaml:"consumer_group" env:"KAFKA_CONSUMER_GROUP" env-default:"booking-service"`
}

type EventService struct {
	BaseURL string `yaml:"base_url" env:"EVENT_SERVICE_URL" env-default:"http://event-service:8082"`

	// HTTP Connection Pool Settings
	MaxIdleConns        int `yaml:"max_idle_conns" env:"HTTP_MAX_IDLE_CONNS" env-default:"20"`
	MaxIdleConnsPerHost int `yaml:"max_idle_conns_per_host" env:"HTTP_MAX_IDLE_CONNS_PER_HOST" env-default:"10"`
	MaxConnsPerHost     int `yaml:"max_conns_per_host" env:"HTTP_MAX_CONNS_PER_HOST" env-default:"20"`
	IdleConnTimeout     int `yaml:"idle_conn_timeout_seconds" env:"HTTP_IDLE_CONN_TIMEOUT" env-default:"90"`
	RequestTimeout      int `yaml:"request_timeout_seconds" env:"HTTP_REQUEST_TIMEOUT" env-default:"30"`
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
