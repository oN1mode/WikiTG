package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

type UserInfo struct {
	userName      string
	userFirstName string
	userLastName  string
	userID        int64
	isBot         bool
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error load .env file: %s", err)
	}

	//Create TG bot
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

	//Connection DB
	connStr := ConfigConnStr()

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

	//Handle messages bot
	for update := range updates {
		if update.Message != nil {
			chatID := update.Message.Chat.ID

			userTG := update.Message.From

			user := UserInfo{
				userName:      userTG.Username,
				userFirstName: userTG.FirstName,
				userLastName:  userTG.LastName,
				userID:        userTG.ID,
				isBot:         userTG.IsBot,
			}

			sendMsg, _ := bot.SendMessage(
				tu.Message(
					tu.ID(chatID),
					"Hello",
				),
			)

			log.Println(sendMsg)
			if ok, err := CheckUserInDB(dbpool, context.Background(), user); !ok {
				if err != nil {
					log.Printf("Error to func CheckUserInDB: %s", err)
				}

				err := InsertUser(dbpool, context.Background(), user)
				if err != nil {
					log.Printf("Error Insert user to main: %s", err)
				}
			}

		}
	}
}

func ConfigConnStr() string {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("HOST"),
		os.Getenv("PORT"),
		os.Getenv("USER"),
		os.Getenv("PASSWORD"),
		os.Getenv("DBNAME"),
		os.Getenv("SSLMODE"),
	)
	return connStr
}

func InsertUser(dbpool *pgxpool.Pool, ctx context.Context, user UserInfo) error {
	tx, err := dbpool.Begin(ctx)
	if err != nil {
		log.Printf("Error to tx dbpool.Begin: %s\n", err)
		return err
	}

	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "INSERT INTO users (tg_name, tg_id, is_bot, first_name, last_name) VALUES ($1, $2, $3, $4, $5)",
		user.userName,
		user.userID,
		user.isBot,
		user.userFirstName,
		user.userLastName,
	)
	if err != nil {
		log.Printf("Error insert to users: %s\n", err)
		return err
	}

	err = tx.Commit(ctx)

	if err != nil {
		log.Printf("Error tx.Commit: %s\n", err)
		return err
	}
	return nil
}

func CheckUserInDB(dbpool *pgxpool.Pool, ctx context.Context, user UserInfo) (bool, error) {
	checkUser := UserInfo{}

	err := dbpool.QueryRow(ctx, "select tg_id from users where tg_id = $1", user.userID).Scan(&checkUser.userID)
	if err != nil {
		log.Printf("Error to select users in CheckUserInDB: %s\n", err)
		return false, err
	}

	return true, nil
}
