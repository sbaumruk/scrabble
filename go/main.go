package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "solve":
			runSolve()
		case "serve":
			runServer()
		case "migrate-boards":
			runMigrateBoards()
		default:
			fmt.Fprintf(os.Stderr, "usage: scrabble [solve|serve|migrate-boards]\n")
			os.Exit(1)
		}
	} else {
		runGame()
	}
}

// runMigrateBoards imports board files from the boards/ directory into PostgreSQL.
// Requires DATABASE_URL to be set. Optionally accepts a user ID as the second argument
// to assign ownership of migrated boards (e.g., ./scrabble migrate-boards <keycloak-sub>).
func runMigrateBoards() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("DATABASE_URL is required for board migration.")
		os.Exit(1)
	}

	ctx := context.Background()
	db, err := NewDB(ctx, dbURL)
	if err != nil {
		fmt.Println("Failed to connect to database:", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		fmt.Println("Failed to run migrations:", err)
		os.Exit(1)
	}

	boardsDir := "boards"
	userID := ""
	if len(os.Args) > 2 {
		userID = os.Args[2]
		fmt.Printf("Migrating boards from %s/ with user_id=%s\n", boardsDir, userID)
	} else {
		fmt.Printf("Migrating boards from %s/ (no user_id, boards will be unowned)\n", boardsDir)
	}

	count, err := db.MigrateBoards(ctx, boardsDir, userID)
	if err != nil {
		fmt.Printf("Migration error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Done. Imported %d board(s).\n", count)
}
