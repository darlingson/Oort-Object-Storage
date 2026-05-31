package main

import (
	"log"
	"net/http"

	"github.com/darlingson/Oort-Object-Storage/internal/api/routes"
	"github.com/darlingson/Oort-Object-Storage/internal/config"
	"github.com/darlingson/Oort-Object-Storage/internal/database"
	"github.com/darlingson/Oort-Object-Storage/internal/database/migrations"
	"github.com/darlingson/Oort-Object-Storage/internal/services"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
	"github.com/darlingson/Oort-Object-Storage/internal/api/handlers"
)

func main() {

	cfg := config.Load()

	db, err := database.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	defer db.Close()

	err = migrations.Run(db, "migrations")
	if err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	repo := repositories.NewPostgresBucketRepository(db)

	bucketService := services.NewBucketService(repo)

	bucketHandler := handlers.NewBucketHandler(
		bucketService,
	)

	router := routes.SetupRoutes(
		bucketHandler,
	)

	log.Printf("server starting on port %s", cfg.AppPort)

	err = http.ListenAndServe(":"+cfg.AppPort, router)
	if err != nil {
		log.Fatalf("server failed: %v", err)
	}
}