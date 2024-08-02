package app

import (
	"WikiBot/internal/bot"
)

func AppRun(configPath string) {
	bot.BotRun(configPath)
}
