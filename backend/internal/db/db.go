package db

import (
	"fmt"

	"cuckoo/backend/internal/config"
	"cuckoo/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Open(cfg config.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.DatabaseDriver {
	case "postgres", "postgresql":
		dialector = postgres.Open(cfg.DatabaseDSN)
	case "sqlite":
		dialector = sqlite.Open(cfg.DatabaseDSN)
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q", cfg.DatabaseDriver)
	}

	conn, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return conn, AutoMigrate(conn)
}

func AutoMigrate(conn *gorm.DB) error {
	return conn.AutoMigrate(
		&models.User{},
		&models.Room{},
		&models.RoomSettings{},
		&models.RoomPlayer{},
		&models.Turn{},
		&models.Contribution{},
		&models.GameResult{},
		&models.GameArchive{},
	)
}
