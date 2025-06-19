package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/amarnathcjd/gogram/telegram"
)

var (
	Db        *sql.DB
	PostLimit int = 5
	Me        *telegram.UserObj
	Config    ConfigType
	StartTime time.Time
	PostText  string = `
<b>Конфа:</b> %s
<b>Количество участников:</b> %v
<b>Категория:</b> %s
<b>Описание:</b>
%s
	`
)

func main() {
	StartTime = time.Now()

	// Парсим файл
	if _, err := toml.DecodeFile("config.toml", &Config); err != nil {
		panic(err)
	}

	Db = DbInit()

	client, err := telegram.NewClient(telegram.ClientConfig{
		AppID: int32(Config.AppID), AppHash: Config.AppHash,
		LogLevel: telegram.LogInfo,
	})
	defer client.Stop()

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	if err := client.Connect(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	client.LoginBot(Config.BotToken) // or client.Login("<phone-number>") for user account, or client.AuthPrompt() for interactive login

	Me, _ = client.GetMe()
	log.Println("Произошел вход на аккаунт " + Me.Username)

	client.On(telegram.OnMessage, func(message *telegram.NewMessage) error {
		if len(message.Message.Entities) > 0 && message.Message.Entities[0].CRC() == 0x6cef8ac7 {
			// пропускаем: это команда, пусть её обрабатывает другой хэндлер
			return nil
		}

		u := GetUserInfo(message.ChatID())
		state := strings.Split(u.State, "|")
		switch state[0] {
		case "admin":
			if state[1] == "edit" {
				admin_conf_edit_finalize(message, u)
			}
			return nil
		case "search":
			if state[1] == "text" {
				search_by_text_final(message)
			}
		}

		return nil

	}, telegram.FilterPrivate)

	client.On("command:start", func(message *telegram.NewMessage) error {
		u := new(UserInfo)
		u.UserID = message.ChatID()
		u.Username = message.Sender.Username
		AddUserInfo(u)

		InsertIntoDB(Db, "UPDATE users SET state = ? WHERE user_id = ?", "idle", u.UserID)

		message.Respond("Привет! Это бот-каталог конф по интересам телеграмма. Каждый может добавить сюда свой кф, чтобы новоприбывшим было легче ориентироваться.", telegram.SendOptions{
			ReplyMarkup: telegram.NewKeyboard().NewGrid(3, 1,
				telegram.Button.Data("🔎 Искать конфы", "search"),
				telegram.Button.Data("⚙️ Мои конфы", "admin"),
				telegram.Button.URL("♿️ Разработчик бота ♿️", "https://t.me/conf_catalog"),
			).Build(),
		})
		return nil
	})

	client.On("callback:start", func(query *telegram.CallbackQuery) error {
		InsertIntoDB(Db, "UPDATE users SET state = ? WHERE user_id = ?", "idle", query.SenderID)

		query.Respond("Привет! Это бот-каталог конф по интересам телеграмма. Каждый может добавить сюда свой кф, чтобы новоприбывшим было легче ориентироваться.", &telegram.SendOptions{
			ReplyMarkup: telegram.NewKeyboard().NewGrid(3, 1,
				telegram.Button.Data("🔎 Искать конфы", "search"),
				telegram.Button.Data("⚙️ Мои конфы", "admin"),
				telegram.Button.URL("♿️ Разработчик бота ♿️", "https://t.me/conf_catalog"),
			).Build(),
		})
		query.Delete()
		return nil
	})

	client.On(telegram.OnParticipant, func(m *telegram.ParticipantUpdate) error {
		if m.User.ID != Me.ID {
			return nil
		}

		if m.IsKicked() || m.IsBanned() || m.IsDemoted() || m.IsLeft() {
			client.SendMessage(
				m.OriginalUpdate.ActorID,
				"😩 Вы выгнали бота из канала. Конфа снята с каталога",
			)

			InsertIntoDB(Db,
				"DELETE FROM admins WHERE channel_id = ?", m.OriginalUpdate.ChannelID)

			InsertIntoDB(Db,
				"DELETE FROM confs WHERE channel_id = ?", m.OriginalUpdate.ChannelID)
			return nil
		}

		insert := InsertIntoDB(Db,
			"INSERT INTO confs (title, post, banner, channel_id, catalog_id) VALUES (?, ?, ?, ?, ?)",
			m.Channel.Title, "", "", m.ChannelID(), 1)
		last_id, _ := insert.LastInsertId()

		InsertIntoDB(Db,
			"INSERT INTO admins (admin_id, channel_id) VALUES (?, ?)",
			m.ActorID(), m.ChannelID())

		client.SendMessage(m.ActorID(), fmt.Sprintf("😘 Бот был добавлен в канал \"%s\"", m.Channel.Title), &telegram.SendOptions{
			ReplyMarkup: telegram.NewKeyboard().NewColumn(1,
				telegram.Button.Data(
					"✍🏻 Редактировать группу",
					"admin|edit|"+strconv.FormatInt(last_id, 10),
				)).Build(),
		})

		return nil
	})

	search_handlers(client)
	admin_handlers(client)
	conf_handler(client)

	client.Idle() // block main goroutine until client is closed
}
