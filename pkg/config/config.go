package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Config structure
type Config struct {
	DownloadDirectory     string `yaml:"download_directory"`
	MaxConcurrentDownloads int    `yaml:"max_concurrent_downloads"`
	SpeedLimitKbps        int    `yaml:"speed_limit_kbps"`
	LogLevel              string `yaml:"log_level"`
}

// LoadConfig reads configuration from a file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	config := &Config{
		DownloadDirectory:     "./downloads",
		MaxConcurrentDownloads: 3,
		SpeedLimitKbps:        0,
		LogLevel:              "info",
	}

	// Read from config file if it exists
	file, err := os.Open(configPath)
	if err == nil {
		defer file.Close()
		decoder := yaml.NewDecoder(file)
		if err := decoder.Decode(config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %v", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to open config file: %v", err)
	}

	// Override from environment variables (if set)
	if val := os.Getenv("DOWNLOAD_DIRECTORY"); val != "" {
		config.DownloadDirectory = val
	}
	if val := os.Getenv("MAX_CONCURRENT_DOWNLOADS"); val != "" {
		fmt.Sscanf(val, "%d", &config.MaxConcurrentDownloads)
	}
	if val := os.Getenv("SPEED_LIMIT_KBPS"); val != "" {
		fmt.Sscanf(val, "%d", &config.SpeedLimitKbps)
	}
	if val := os.Getenv("LOG_LEVEL"); val != "" {
		config.LogLevel = val
	}

	return config, nil
}

// PrintConfig logs the loaded configuration
func PrintConfig(config *Config) {
	log.Printf("Configuration Loaded:\n"+
		"- Download Directory: %s\n"+
		"- Max Concurrent Downloads: %d\n"+
		"- Speed Limit (KBps): %d\n"+
		"- Log Level: %s\n",
		config.DownloadDirectory,
		config.MaxConcurrentDownloads,
		config.SpeedLimitKbps,
		config.LogLevel,
	)
}