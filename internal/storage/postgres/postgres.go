package postgres

import (
	"WikiBot/internal/model"
	"WikiBot/internal/utils"
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)


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

		url = utils.RecoveryString(url)

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

	log.Printf("String Trim: %s\n", utils.TrimString(response))

	_, err = tx.Exec(ctx, "INSERT INTO response_api_history (url_response, created_at, tg_id_usr) VALUES ($1, $2, $3)",
		utils.TrimString(response),
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

	defer rows.Close()

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

func InsertUser(dbpool *pgxpool.Pool, ctx context.Context, user model.UserInfo) error {
	tx, err := dbpool.Begin(ctx)
	if err != nil {
		log.Printf("Error to tx dbpool.Begin: %s\n", err)
		return err
	}

	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, "INSERT INTO users (tg_name, tg_id, is_bot, first_name, last_name, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
		user.GetName(),
		user.GetID(),
		user.IsBot(),
		user.GetFirstName(),
		user.GetLastName(),
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

func CheckUserInDB(dbpool *pgxpool.Pool, ctx context.Context, user model.UserInfo) (bool, error) {
	checkUser := model.UserInfo{}

	err := dbpool.QueryRow(ctx, "SELECT tg_id FROM users WHERE tg_id = $1", user.GetID()).Scan(&checkUser.Id)
	if err != nil {
		log.Printf("Error to select users in CheckUserInDB: %s\n", err)
		return false, err
	}

	return true, nil
}
