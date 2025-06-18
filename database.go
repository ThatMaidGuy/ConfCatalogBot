package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func DbInit() *sql.DB {
	// Открываем базу данных
	db, err := sql.Open("mysql", Config.DBName)
	if err != nil {
		log.Fatal(err)
	}

	// Проверяем подключение
	err = db.Ping()
	if err != nil {
		log.Fatal("Ping error:", err)
	}

	fmt.Println("База данных успешно инициализирована.")
	return db
}

func InsertIntoDB(db *sql.DB, query string, args ...any) sql.Result {
	stmt, err := db.Prepare(query)
	if err != nil {
		panic(err)
	}
	result, err := stmt.Exec(args...)
	if err != nil {
		panic(err)
	}
	return result
}

func GetUserInfo(user_id int64) *UserInfo {
	result := new(UserInfo)
	res := Db.QueryRow("SELECT * FROM users WHERE user_id = ?", user_id)
	err := res.Scan(&result.ID, &result.UserID, &result.Username, &result.State)
	if err != nil {
		return nil
	}
	return result
}

func HasUserInfo(u *UserInfo) bool {
	count := 0

	q := "SELECT count(*) FROM users WHERE user_id = ?"

	res := Db.QueryRow(q, u.UserID)
	err := res.Scan(&count)
	if err != nil {
		return count != 0
	}

	return count != 0
}

func AddUserInfo(u *UserInfo) {
	if HasUserInfo(u) {
		return
	}

	InsertIntoDB(Db, "INSERT INTO users (user_id, username, state) VALUES (?, ?, ?)", u.UserID, u.Username, "idle")
}
