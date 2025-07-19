package config

import (
	"github.com/ilyakaznacheev/cleanenv"
)

var configuration Config

type Config struct {
	Port      string         `yaml:"port" env:"PORT"`
	Database  DatabaseConfig `yaml:"database" env:"DATABASE"`
	JWTSecret string         `yaml:"jwt_secret" env:"JWT_SECRET"`
}

type DatabaseConfig struct {
	User         string `yaml:"user" env:"DB_USER"`
	Password     string `yaml:"password" env:"DB_PASSWORD"`
	DatabaseName string `yaml:"database_name" env:"DB_NAME"`
	Host         string `yaml:"host" env:"DB_HOST"`
	Port         string `yaml:"port" env:"DB_PORT"`
	SSLMode      string `yaml:"ssl_mode" env:"DB_SSL_MODE"`
}

// GetDatabaseURL constructs the PostgreSQL connection string
func (d *DatabaseConfig) GetDatabaseURL() string {
	return "postgres://" + d.User + ":" + d.Password + "@" + d.Host + ":" + d.Port + "/" + d.DatabaseName + "?sslmode=" + d.SSLMode
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
		configuration.Port = "8081"
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

	return &configuration, nil
}

func GetConfig() *Config {
	return &configuration
}
