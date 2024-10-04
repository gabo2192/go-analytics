package config

import "os"

type Config struct {
	Host     string
	Port     string
	Password string
	User     string
	SSLMode  string
	DBName   string
}

func GetConfig() *Config {
	return &Config{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Password: os.Getenv("DB_PASS"),
		User:     os.Getenv("DB_USER"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
		DBName:   os.Getenv("DB_NAME"),
	}
}