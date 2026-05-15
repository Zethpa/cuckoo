package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"cuckoo/backend/internal/config"
	"cuckoo/backend/internal/db"
	"cuckoo/backend/internal/services"
)

func main() {
	if len(os.Args) < 3 || os.Args[1] != "user" || os.Args[2] != "add" {
		fmt.Println("usage: cuckoo user add --username alice [--password secret123] --role player")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("user add", flag.ExitOnError)
	username := fs.String("username", "", "username")
	password := fs.String("password", "", "password")
	role := fs.String("role", "player", "role: admin or player")
	_ = fs.Parse(os.Args[3:])

	cfg := config.Load()
	conn, err := db.Open(cfg)
	if err != nil {
		log.Fatal(err)
	}
	authSvc := services.NewAuthService(conn, cfg)
	if *password == "" {
		generated, err := authSvc.AddUserWithGeneratedPassword(*username, *role)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("created user %s with initial password: %s\n", *username, generated)
		return
	}
	if err := authSvc.AddUser(*username, *password, *role); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("created user %s\n", *username)
}
