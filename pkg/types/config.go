package types

import (
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"log"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	OpenAI   OpenAIConfig
	Gemini   GeminiConfig
}

type ServerConfig struct {
	Host            string
	Port            string
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
	AppEnv          string
	LogLevel        string
}

type DatabaseConfig struct {
	Name     string
	Host     string
	Port     string
	User     string
	Password string
	SSLMode  string
}

type OpenAIConfig struct {
	APIKey string
}

type GeminiConfig struct {
	APIKey string
}

func validateRequiredEnvs(v *viper.Viper, requiredEnvs []string) error {
	for _, env := range requiredEnvs {
		if v.GetString(env) == "" {
			return fmt.Errorf("%s is required", env)
		}
	}
	return nil
}

// LoadConfig reads configuration from environment variables
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Enable environment variable reading first
	v.AutomaticEnv()

	v.SetConfigFile(".env")
	v.SetConfigType("env")
	if err := v.ReadInConfig(); err != nil {
		log.Print("No config file found, falling back to environment variables")
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, err
		}
	}

	requiredEnvs := []string{
		"DB_NAME",
		"DB_HOST",
		"DB_PORT",
		"DB_USER",
		"DB_PASSWORD",
		"DB_SSLMODE",
	}

	if err := validateRequiredEnvs(v, requiredEnvs); err != nil {
		return nil, err
	}

	config := &Config{
		Server: ServerConfig{
			Host:     v.GetString("SERVER_HOST"),
			Port:     v.GetString("SERVER_PORT"),
			AppEnv:   v.GetString("APP_ENV"),
			LogLevel: v.GetString("LOG_LEVEL"),
		},
		Database: DatabaseConfig{
			Name:     v.GetString("DB_NAME"),
			Host:     v.GetString("DB_HOST"),
			Port:     v.GetString("DB_PORT"),
			User:     v.GetString("DB_USER"),
			Password: v.GetString("DB_PASSWORD"),
			SSLMode:  v.GetString("DB_SSLMODE"),
		},
		OpenAI: OpenAIConfig{
			APIKey: v.GetString("OPENAI_API_KEY"),
		},
		Gemini: GeminiConfig{
			APIKey: v.GetString("GEMINI_API_KEY"),
		},
	}

	// Set default values for server if not provided
	if config.Server.Port == "" {
		config.Server.Port = "6777"
	}

	return config, nil
}

// GetServerAddress returns the full server address
func (c *ServerConfig) GetServerAddress() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}
