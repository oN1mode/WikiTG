package bot

import (
	"WikiBot/config"
	"WikiBot/internal/api"
	"WikiBot/internal/model"
	"WikiBot/internal/storage/postgres"
	"WikiBot/internal/utils"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

func BotRun(configPath string) {
	err := godotenv.Load(configPath)
	if err != nil {
		log.Fatalf("Error load .env file: %s", err)
	}

	//Создаем бота
	botToken := os.Getenv("BOT_API_TOKEN")

	bot, err := telego.NewBot(botToken, telego.WithDefaultDebugLogger())
	if err != nil {
		log.Fatalf("Error init new bot: %s", err)
	}

	updates, err := bot.UpdatesViaLongPolling(nil)
	if err != nil {
		log.Printf("Error bot polling: %s\n", err)
	}

	defer bot.StopLongPolling()

	//Подключаемся к БД
	connStr := config.ConfigConnStr()

	dbpool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Error created db pool: %s", err)
	}

	defer dbpool.Close()

	err = dbpool.Ping(context.Background())
	if err != nil {
		log.Fatalf("Error db pool Ping: %s", err)
	}

	log.Println("Bot started, db connection")

	//Обрабатываем сообщения от бота
	for update := range updates {
		if update.Message != nil {
			chatID := update.Message.Chat.ID

			userTG := update.Message.From

			user := model.UserInfo{}.InitUser(userTG.Username, userTG.FirstName, userTG.LastName, userTG.ID, userTG.IsBot)

			if ok, err := postgres.CheckUserInDB(dbpool, context.Background(), user); !ok {
				if err != nil {
					log.Printf("Error to func CheckUserInDB: %s", err)
				}

				err := postgres.InsertUser(dbpool, context.Background(), user)
				if err != nil {
					log.Printf("Error Insert user to main: %s", err)
				}
			}

			switch update.Message.Text {
			case "/start":
				bot.SendMessage(
					tu.Message(
						tu.ID(chatID),
						"Привет! Этот телеграм бот поможет отыскать интересующие статьи в Wikipedia. Для получения статей, напиши интересующую тему.",
					),
				)

			case "/info":
				bot.SendMessage(
					tu.Message(
						tu.ID(chatID),
						`Этот телеграм бот поможет отыскать интересующие статьи в Wikipedia. Для получения статей, напиши интересующую тему.
						Так же имеются комманды /request-history и /response-history.
						Команда /request-history вернет информацию о последних трех запросах.
						Команда /response-history вернёт информацию о последних трех полученных ответов`,
					),
				)

			case "/request-history":
				reqHis, err := postgres.GetRequestHistory(dbpool, context.Background(), user.GetID())
				if err != nil {
					log.Printf("Error to select request history: %s", err)
				}

				bot.SendMessage(
					tu.Message(
						tu.ID(chatID),
						"Последние три запроса:",
					),
				)

				count := 1
				for _, val := range reqHis {
					bot.SendMessage(
						tu.Message(
							tu.ID(chatID),
							fmt.Sprintf("Запрос № %v -> %s", count, val),
						),
					)
					count++
				}

			case "/response-history":
				resHis, err := postgres.GetResponseHistory(dbpool, context.Background(), user.GetID())
				if err != nil {
					log.Printf("Error to select response history: %s", err)
				}

				bot.SendMessage(
					tu.Message(
						tu.ID(chatID),
						"Последние три полученных ответа:",
					),
				)

				count := 1
				for _, val := range resHis {
					bot.SendMessage(
						tu.Message(
							tu.ID(chatID),
							fmt.Sprintf("Ответ № %v -> %s", count, val),
						),
					)
					count++
				}

			default:
				//Устанавливаем язык для поиска в Википедии
				language := os.Getenv("LANGUAGE")

				//Создаем url для поиска
				url, _ := utils.UrlEncoded(update.Message.Text)

				request := "https://" + language + ".wikipedia.org/w/api.php?action=opensearch&search=" + url + "&limit=3&origin=*&format=json"

				//Присваем данные среза с ответом в переменную message
				message := api.WikipediaAPI(request)

				if message[0] != "Что-то пошло не так, попробуйте изменить вопрос." {
					postgres.InsertRequestUser(dbpool, context.Background(), update.Message.Text, user.GetID())
				}

				for _, val := range message {
					//Отправлем сообщение
					bot.SendMessage(
						tu.Message(
							tu.ID(chatID),
							val,
						),
					)
					err = postgres.InsertResponseHistory(dbpool, context.Background(), val, user.GetID())
					if err != nil {
						log.Printf("Error to insert resposne history: %s\n", err)
					}
				}
			}
		}
	}
}
