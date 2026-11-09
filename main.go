package main

import (
	"log/slog"
	"os"

	"github.com/mmfshirokan/task-manager/app"
)

func main() {
	if err := app.Run(); err != nil {
		slog.Error("service stopped with error", "error", err)
		os.Exit(1)
	}
}
