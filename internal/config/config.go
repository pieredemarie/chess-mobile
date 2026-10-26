package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port        int
	DatabaseURL string
	JWTSecret   string
}

func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}
	return Parse(os.Getenv("PORT"), os.Getenv("DATABASE_URL"), os.Getenv("JWT_SECRET"))
}

// loadDotEnv fills missing process variables for local runs. Injected variables take priority.
func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	for line := 1; scanner.Scan(); line++ {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" || strings.HasPrefix(raw, "#") {
			continue
		}
		key, value, ok := strings.Cut(raw, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" || strings.ContainsAny(key, " \t") {
			return fmt.Errorf("invalid %s line %d", path, line)
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err = os.Setenv(key, value); err != nil {
				return fmt.Errorf("set %s: %w", key, err)
			}
		}
	}
	if err = scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}

func Parse(port, dsn, secret string) (Config, error) {
	if strings.TrimSpace(port) == "" {
		return Config{}, errors.New("PORT is required")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return Config{}, errors.New("PORT must be an integer from 1 to 65535")
	}
	if strings.TrimSpace(dsn) == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if strings.TrimSpace(secret) == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}
	return Config{Port: n, DatabaseURL: dsn, JWTSecret: secret}, nil
}
func (c Config) Address() string { return fmt.Sprintf(":%d", c.Port) }
