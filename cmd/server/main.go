package main

import (
	"log"
	"net/http"

	"durpdeploy/internal/migrate"
	"durpdeploy/internal/repository"
	"durpdeploy/internal/runner"
	"durpdeploy/internal/server"
)

func main() {
	dsn := "durpdeploy.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"

	db, err := migrate.Run(dsn)
	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	defer db.Close()
	log.Println("database ready")

	repo := repository.New(db)
	broker := runner.NewLogBroker()
	rnr := runner.New(repo, broker)
	r := server.NewRouter(repo, rnr)
	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
