package config

import (
	"github.com/Juuwe/data-migration-backend/internal/database"
)

type Config struct {
	WebServer struct {
		Host string
		Port int
	}

	DB database.Config
}
