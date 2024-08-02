package main

import (
	"WikiBot/internal/app"
)

const configPath = "config/.env"

func main() {
	app.AppRun(configPath)
}
