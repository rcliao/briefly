package main

import (
	"briefly/cmd/handlers"
	"briefly/internal/logger"
	"fmt"
	"os"
)

func main() {
	logger.Init() // Initialize the logger

	// Use simplified command structure
	if err := handlers.ExecuteSimplified(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
