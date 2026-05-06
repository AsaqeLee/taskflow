package main

import (
	"log"

	"github.com/AsaqeLee/taskflow/internal/bootstrap"
	"github.com/AsaqeLee/taskflow/internal/config"
)

func main() {
	cfg := config.Load()
	app := bootstrap.NewApp(cfg)

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
