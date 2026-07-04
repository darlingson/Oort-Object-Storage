package main

import (
	"context"
	"log"
	"net/http"

	"github.com/darlingson/Oort-Object-Storage/internal/api/handlers"
	"github.com/darlingson/Oort-Object-Storage/internal/api/routes"
	"github.com/darlingson/Oort-Object-Storage/internal/config"
	"github.com/darlingson/Oort-Object-Storage/internal/database"
	"github.com/darlingson/Oort-Object-Storage/internal/database/migrations"
	"github.com/darlingson/Oort-Object-Storage/internal/services"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/filesystem"
	"github.com/darlingson/Oort-Object-Storage/internal/storage/repositories"
)

func main() {

	cfg := config.Load()
	config.InitLogger()

	db, err := database.NewPostgres(cfg)
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	defer db.Close()

	err = migrations.Run(db, "migrations")
	if err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	permissionRepo := repositories.NewPostgresPermissionRepository(db)
	roleRepo := repositories.NewPostgresRoleRepository(db)
	userRepo := repositories.NewPostgresUserRepository(db)

	seedSvc := services.NewSeedService(
		permissionRepo,
		roleRepo,
		userRepo,
	)
	seedSvc.SeedIfNeeded(context.Background())

	bucketRepo := repositories.NewPostgresBucketRepository(db)
	bucketService := services.NewBucketService(bucketRepo)
	bucketHandler := handlers.NewBucketHandler(bucketService)

	filesystemDriver := filesystem.NewLocalDriver("./data/blobs")
	objectRepo := repositories.NewPostgresObjectRepository(db)
	objectService := services.NewObjectService(
		bucketRepo,
		objectRepo,
		filesystemDriver,
	)
	objectHandler := handlers.NewObjectHandler(objectService)

	keyRepo := repositories.NewPostgresKeyRepository(db)
	keyService := services.NewKeyService(keyRepo)
	keyHandler := handlers.NewKeyHandler(keyService)

	userService := services.NewUserService(userRepo)
	jwtService := services.NewJWTService()

	authHandler := handlers.NewAuthHandler(userService, jwtService)
	userHandler := handlers.NewUserHandler(userService, permissionRepo)

	healthService := services.NewHealthService(db, "./data/blobs")
	healthHandler := handlers.NewHealthHandler(healthService)

	router := routes.SetupRoutes(
		bucketHandler,
		objectHandler,
		keyHandler,
		authHandler,
		userHandler,
		healthHandler,
		keyService,
		jwtService,
	)

	log.Printf("server starting on port %s", cfg.AppPort)

	err = http.ListenAndServe(":"+cfg.AppPort, router)
	if err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
