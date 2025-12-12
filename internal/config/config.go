package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Logger      *log.Logger
	PostgresURL string `env:"PostgresUrl"`

	// Redis struct {
	// 	Addr     string `yaml:"Addr"`
	// 	Password string `yaml:"Password"`
	// 	DB       int    `yaml:"DB"`
	// } `yaml:"Redis"`
}

var (
	instance *Config
	once     sync.Once
)

func InitConfig(logPrefix string) (*Config, error) {
	var initErr error
	once.Do(func() {
		instance, initErr = initializeConfig(logPrefix)
	})
	return instance, initErr
}

func initializeConfig(logPrefix string) (*Config, error) {
	logDir := "logs"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		os.Mkdir(logDir, 0755)
	}

	// Создаем файл для логов аутентификации
	logFile, err := os.OpenFile(
		filepath.Join("logs", fmt.Sprintf("%s.log", logPrefix)),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0666,
	)

	if err != nil {
		log.Fatalf("couldn't open log file: %v", err)
	}

	configPaths := []string{
		"../../config.env",
		"config.env",
		"./config.env",
		"../config.env",
	}

	var foundPath string
	for _, path := range configPaths {
		if _, err := os.Stat(path); err == nil {
			foundPath = path
			break
		}
	}

	if foundPath == "" {
		return nil, fmt.Errorf("failed to find config.env in any of these paths: %v", configPaths)
	}

	fmt.Printf("Config found at: %s\n", foundPath)

	err = godotenv.Load(foundPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file from %s: %v", foundPath, err)
	}

	var config Config
	err = env.Parse(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %v", err)
	}

	config.Logger = log.New(logFile, logPrefix, log.Ldate|log.Ltime)
	return &config, nil
}
