package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

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

type SearchResults struct {
	ready   bool
	Query   string
	Results []Result
}

type Result struct {
	Name, Description, URL string
}

func main() {
	err := godotenv.Load()
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

	//Обрабатываем сообщения от бота
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

			if ok, err := CheckUserInDB(dbpool, context.Background(), user); !ok {
				if err != nil {
					log.Printf("Error to func CheckUserInDB: %s", err)
				}

				err := InsertUser(dbpool, context.Background(), user)
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
				reqHis, err := GetRequestHistory(dbpool, context.Background(), user.userID)
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
				resHis, err := GetResponseHistory(dbpool, context.Background(), user.userID)
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
				url, _ := urlEncoded(update.Message.Text)

				request := "https://" + language + ".wikipedia.org/w/api.php?action=opensearch&search=" + url + "&limit=3&origin=*&format=json"

				//Присваем данные среза с ответом в переменную message
				message := wikipediaAPI(request)

				if message[0] != "Что-то пошло не так, попробуйте изменить вопрос." {
					InsertRequestUser(dbpool, context.Background(), update.Message.Text, user.userID)
				}

				for _, val := range message {
					//Отправлем сообщение
					bot.SendMessage(
						tu.Message(
							tu.ID(chatID),
							val,
						),
					)
					InsertResponseHistory(dbpool, context.Background(), val, user.userID)
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

func TrimString(str string) string {
	return strings.Trim(str, "https://ru.wikipedia.org/wiki/")
}

func RecoveryString(str string) string {
	return ("https://ru.wikipedia.org/wiki/" + str)
}

func GetResponseHistory(dbpool *pgxpool.Pool, ctx context.Context, userID int64) ([]string, error) {
	resHis := make([]string, 0, 3)

	rows, err := dbpool.Query(ctx, "SELECT url_response FROM response_api_history WHERE tg_id_usr = $1 ORDER BY created_at DESC LIMIT 3", userID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	for rows.Next() {
		var url string
		err := rows.Scan(&url)
		if err != nil {
			return nil, err
		}

		url = RecoveryString(url)

		resHis = append(resHis, url)
	}

	return resHis, nil
}

func InsertResponseHistory(dbpool *pgxpool.Pool, ctx context.Context, response string, userID int64) error {
	tx, err := dbpool.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "INSERT INTO resposne_api_history (url_response, created_at, tg_id_usr) VALUES ($1, $2, $3)",
		TrimString(response),
		time.Now(),
		userID,
	)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func GetRequestHistory(dbpool *pgxpool.Pool, ctx context.Context, userID int64) ([]string, error) {
	reqHis := make([]string, 0, 3)

	rows, err := dbpool.Query(ctx, "SELECT text_request FROM user_request WHERE tg_id_usr = $1 ORDER BY created_at DESC LIMIT 3", userID)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var text string
		err := rows.Scan(&text)
		if err != nil {
			return nil, err
		}

		reqHis = append(reqHis, text)
	}

	return reqHis, nil
}

func InsertRequestUser(dbpool *pgxpool.Pool, ctx context.Context, request string, userID int64) error {
	tx, err := dbpool.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "INSERT INTO user_request (text_request, created_at, tg_id_usr) VALUES ($1, $2, $3)",
		request,
		time.Now(),
		userID,
	)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func InsertUser(dbpool *pgxpool.Pool, ctx context.Context, user UserInfo) error {
	tx, err := dbpool.Begin(ctx)
	if err != nil {
		log.Printf("Error to tx dbpool.Begin: %s\n", err)
		return err
	}

	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "INSERT INTO users (tg_name, tg_id, is_bot, first_name, last_name, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
		user.userName,
		user.userID,
		user.isBot,
		user.userFirstName,
		user.userLastName,
		time.Now(),
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

	err := dbpool.QueryRow(ctx, "SELECT tg_id FROM users WHERE tg_id = $1", user.userID).Scan(&checkUser.userID)
	if err != nil {
		log.Printf("Error to select users in CheckUserInDB: %s\n", err)
		return false, err
	}

	return true, nil
}

func (sr *SearchResults) UnmarshalJSON(bs []byte) error {
	array := []interface{}{}
	if err := json.Unmarshal(bs, &array); err != nil {
		return err
	}
	sr.Query = array[0].(string)
	for i := range array[1].([]interface{}) {
		sr.Results = append(sr.Results, Result{
			array[1].([]interface{})[i].(string),
			array[2].([]interface{})[i].(string),
			array[3].([]interface{})[i].(string),
		})
	}
	return nil
}

func urlEncoded(str string) (string, error) {
	u, err := url.Parse(str)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func wikipediaAPI(request string) (answer []string) {

	//Создаем срез на 3 элемента
	s := make([]string, 3)

	//Отправляем запрос
	if response, err := http.Get(request); err != nil {
		s[0] = "Википедия не отвечает"
	} else {
		defer response.Body.Close()

		//Считываем ответ
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Printf("Error to read content: %s\n", err)
		}

		//Отправляем данные в структуру
		sr := &SearchResults{}
		if err = json.Unmarshal([]byte(contents), sr); err != nil {
			s[0] = "Что-то пошло не так, попробуйте изменить вопрос."
		}

		//Проверяем не пустая ли наша структура
		if !sr.ready {
			s[0] = "Что-то пошло не так, попробуйте изменить вопрос."
		}

		//Проходим через нашу структуру и отправляем данные в срез с ответом
		for i := range sr.Results {
			s[i] = sr.Results[i].URL
		}
	}

	return s
}
