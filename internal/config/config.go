package config

import (
	"bufio"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	WebServer WebServer
	Database  Database
	MinIO     MinIO
}

type WebServer struct {
	Host string
	Port int
}

func (c WebServer) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type Database struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
	Timezone string
}

func (c Database) DSN() string {
	query := url.Values{
		"sslmode":  []string{c.SSLMode},
		"TimeZone": []string{c.Timezone},
	}

	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(c.User, c.Password),
		Host:     fmt.Sprintf("%s:%d", c.Host, c.Port),
		Path:     c.Name,
		RawQuery: query.Encode(),
	}).String()
}

type MinIO struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	BaseURL   string
	UseSSL    bool
}

func Load(dotEnvPath string) (Config, error) {
	if err := loadDotEnv(dotEnvPath); err != nil {
		return Config{}, err
	}

	webPort, err := envInt("WEB_PORT", 8081)
	if err != nil {
		return Config{}, err
	}
	dbPort, err := envInt("DB_PORT", 5432)
	if err != nil {
		return Config{}, err
	}
	minioUseSSL, err := envBool("MINIO_USE_SSL", false)
	if err != nil {
		return Config{}, err
	}

	return Config{
		WebServer: WebServer{
			Host: envString("WEB_HOST", "0.0.0.0"),
			Port: webPort,
		},
		Database: Database{
			Host:     envString("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     envString("DB_USER", "postgres"),
			Password: envString("DB_PASSWORD", "postgres"),
			Name:     envString("DB_NAME", "data_migration"),
			SSLMode:  envString("DB_SSL_MODE", "disable"),
			Timezone: envString("DB_TIMEZONE", "Europe/Moscow"),
		},
		MinIO: MinIO{
			Endpoint:  envString("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: envString("MINIO_ACCESS_KEY", "admin"),
			SecretKey: envString("MINIO_SECRET_KEY", "password123"),
			Bucket:    envString("MINIO_BUCKET", "data-migration-service"),
			BaseURL:   envString("MINIO_BASE_URL", "http://localhost:9000"),
			UseSSL:    minioUseSSL,
		},
	}, nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open environment file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(strings.TrimPrefix(line, "export "), "=")
		key = strings.TrimSpace(key)
		if !found || key == "" {
			return fmt.Errorf("invalid environment line %q", line)
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		value = strings.Trim(strings.TrimSpace(value), "\"'")
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set environment variable %s: %w", key, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read environment file: %w", err)
	}

	return nil
}

func envString(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) (int, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 || parsed > 65535 {
		return 0, fmt.Errorf("%s must be a valid port", key)
	}
	return parsed, nil
}

func envBool(key string, fallback bool) (bool, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean: %w", key, err)
	}
	return parsed, nil
}
