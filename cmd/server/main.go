package main

import (
	"log"
	"net/http"

	"github.com/darlingson/Oort-Object-Storage/internal/api/routes"
	"github.com/darlingson/Oort-Object-Storage/internal/config"
	"github.com/darlingson/Oort-Object-Storage/internal/database"
)

func main() {

	cfg := config.Load()

	db, err := database.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	defer db.Close()

	router := routes.SetupRoutes()

	log.Printf("server starting on port %s", cfg.AppPort)

	err = http.ListenAndServe(":"+cfg.AppPort, router)
	if err != nil {
		log.Fatalf("server failed: %v", err)
	}
}