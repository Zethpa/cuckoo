package main

import (
	"log"

	"cuckoo/backend/internal/config"
	"cuckoo/backend/internal/db"
	"cuckoo/backend/internal/handlers"
	"cuckoo/backend/internal/realtime"
	"cuckoo/backend/internal/services"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	conn, err := db.Open(cfg)
	if err != nil {
		log.Fatal(err)
	}

	authSvc := services.NewAuthService(conn, cfg)
	if err := authSvc.SeedAdmin(); err != nil {
		log.Fatal(err)
	}

	hub := realtime.NewHub()
	roomSvc := services.NewRoomService(conn, hub)
	handler := handlers.New(cfg, authSvc, roomSvc)

	r := gin.Default()
	handler.Register(r, hub.Handler(cfg))
	log.Printf("cuckoo backend listening on %s", cfg.HTTPAddr)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
