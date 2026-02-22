package config

import (
	"fmt"
	"log"
	"ms-ride-sharing/services/user-service/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB(config *Config) *gorm.DB {
	POSTGRES_DSN := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.PostgresHost,
		config.PostgresPort,
		config.PostgresUser,
		config.PostgresPassword,
		config.PostgresDB,
	)

	db, err := gorm.Open(postgres.Open(POSTGRES_DSN), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to start connection with database:", err)
	}

	if err := db.AutoMigrate(
		&models.User{},
	); err != nil {
		log.Fatal("failed to run the auto migrate process:", err)
	}

	log.Println("database migration completed successfully!")

	return db
}
