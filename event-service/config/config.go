package config

import (
	"github.com/ilyakaznacheev/cleanenv"
)

var configuration Config

type Config struct {
	Port      string         `yaml:"port" env:"PORT"`
	Database  DatabaseConfig `yaml:"database" env:"DATABASE"`
	JWTSecret string         `yaml:"jwt_secret" env:"JWT_SECRET"`
	Redis     RedisConfig    `yaml:"redis" env:"REDIS"`
}

type DatabaseConfig struct {
	User         string `yaml:"user" env:"DB_USER"`
	Password     string `yaml:"password" env:"DB_PASSWORD"`
	DatabaseName string `yaml:"database_name" env:"DB_NAME"`
	Host         string `yaml:"host" env:"DB_HOST"`
	Port         string `yaml:"port" env:"DB_PORT"`
	SSLMode      string `yaml:"ssl_mode" env:"DB_SSL_MODE"`
}

type RedisConfig struct {
	Host     string `yaml:"host" env:"REDIS_HOST"`
	Port     string `yaml:"port" env:"REDIS_PORT"`
	Password string `yaml:"password" env:"REDIS_PASSWORD"`
	DB       int    `yaml:"db" env:"REDIS_DB"`
}

// GetDatabaseURL constructs the PostgreSQL connection string
func (d *DatabaseConfig) GetDatabaseURL() string {
	return "postgres://" + d.User + ":" + d.Password + "@" + d.Host + ":" + d.Port + "/" + d.DatabaseName + "?sslmode=" + d.SSLMode
}

// GetRedisURL constructs the Redis connection string
func (r *RedisConfig) GetRedisURL() string {
	return r.Host + ":" + r.Port
}

func Initialise(filepath string, env bool) (*Config, error) {
	var err error

	if env {
		err = cleanenv.ReadEnv(&configuration)
	} else {
		err = cleanenv.ReadConfig(filepath, &configuration)
	}

	if err != nil {
		return nil, err
	}

	// Set defaults if not provided
	if configuration.Port == "" {
		configuration.Port = "8082"
	}
	if configuration.Database.User == "" {
		configuration.Database.User = "postgres"
	}
	if configuration.Database.Password == "" {
		configuration.Database.Password = "password"
	}
	if configuration.Database.DatabaseName == "" {
		configuration.Database.DatabaseName = "eventbooking"
	}
	if configuration.Database.Host == "" {
		configuration.Database.Host = "localhost"
	}
	if configuration.Database.Port == "" {
		configuration.Database.Port = "5432"
	}
	if configuration.Database.SSLMode == "" {
		configuration.Database.SSLMode = "disable"
	}
	if configuration.JWTSecret == "" {
		configuration.JWTSecret = "your-secret-key-change-in-production"
	}
	if configuration.Redis.Host == "" {
		configuration.Redis.Host = "localhost"
	}
	if configuration.Redis.Port == "" {
		configuration.Redis.Port = "6379"
	}
	if configuration.Redis.DB == 0 {
		configuration.Redis.DB = 0
	}

	return &configuration, nil
}

func GetConfig() *Config {
	return &configuration
}
