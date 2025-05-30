package main

import (
	"briefly/cmd/cmd"
	"briefly/internal/logger"
)

func main() {
	logger.Init() // Initialize the logger
	cmd.Execute()
}
