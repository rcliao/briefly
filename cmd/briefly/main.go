package main

import (
	"briefly/cmd/handlers"
	"briefly/internal/logger"
)

func main() {
	logger.Init() // Initialize the logger
	handlers.Execute()
}
