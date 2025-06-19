package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func admin_handlers(client *telegram.Client) {
	client.On("callback:admin", func(query *telegram.CallbackQuery) error {
		args := strings.Split(string(query.Data), "|")
		if len(args) > 1 {
			switch args[1] {
			case "add":
				admin_conf_add(query)
				return nil
			case "edit":
				admin_conf_edit(client, query)
				return nil
			case "list":
				admin_conf_list(query)
				return nil
			}
		}

		InsertIntoDB(Db, "UPDATE users SET state = ? WHERE user_id = ?", "idle", query.SenderID)

		query.Edit("Админ конфы? Добавь её к нам!", &telegram.SendOptions{
			ReplyMarkup: telegram.NewKeyboard().NewGrid(3, 1,
				telegram.Button.Data("🎁 Добавить конфу", "admin|add"),
				telegram.Button.Data("Мои конфы", "admin|list"),
				telegram.Button.Data("↪️ Обратно", "start"),
			).Build(),
		})
		return nil
	})
}

func admin_conf_add(query *telegram.CallbackQuery) {
	query.Edit("Чтобы добавить конфу в наш каталог, вам необходимо добавить нашего бота в ваш канал. Из прав на данный момент достаточны лишь права \"Добавление подписчиков\"", &telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().NewGrid(1, 1,
			telegram.Button.Data("↪️ Обратно", "start"),
		).Build(),
	})
}

func admin_conf_edit(client *telegram.Client, query *telegram.CallbackQuery) {
	InsertIntoDB(Db, "UPDATE users SET state = ? WHERE user_id = ?", "idle", query.SenderID)

	args := strings.Split(string(query.Data), "|")
	if len(args) < 3 {
		query.Edit("Ошибка")
		return
	}
	id, _ := strconv.Atoi(args[2])

	conf := new(ConfInfo)

	res := Db.QueryRow("SELECT * FROM confs WHERE id = ?", id)
	err := res.Scan(&conf.ID, &conf.Title, &conf.Post, &conf.Banner, &conf.ChannelID, &conf.CatalogID)
	if err != nil {
		query.Edit("Ошибка: " + err.Error())
		return
	}

	if len(args) > 3 {
		switch args[3] {
		case "title":
			admin_conf_edit_title(query, conf)
		case "banner":
			admin_conf_edit_banner(query, conf)
		case "desc":
			admin_conf_edit_desc(query, conf)
		case "cat":
			admin_conf_edit_cat(query, conf)
		}
		return
	}

	var cat_title string
	res = Db.QueryRow("SELECT name FROM catalogs WHERE id = ?", conf.CatalogID)
	err = res.Scan(&cat_title)
	if err != nil {
		query.Edit("Ошибка: " + err.Error())
		return
	}

	channel, err := client.GetChannel(conf.ChannelID)
	if err != nil {
		log.Println(err)
		return
	}
	participants := channel.ParticipantsCount

	strid := strconv.FormatInt(conf.ID, 10)
	kb := telegram.NewKeyboard().NewGrid(3, 2,
		telegram.Button.Data("Изменить название", "admin|edit|"+strid+"|title"),
		telegram.Button.Data("Изменить баннер", "admin|edit|"+strid+"|banner"),
		telegram.Button.Data("Изменить описание", "admin|edit|"+strid+"|desc"),
		telegram.Button.Data("Изменить категорию", "admin|edit|"+strid+"|cat"),
	).NewGrid(2, 1,
		telegram.Button.Data("↪️ Обратно", "start"),
	).Build()

	if conf.Banner == "" {
		query.Respond(fmt.Sprintf(PostText, conf.Title, participants, cat_title, conf.Post), &telegram.SendOptions{
			ParseMode:   "html",
			ReplyMarkup: kb,
		})
		query.Delete()
		return
	}
	banner, _ := telegram.ResolveBotFileID(conf.Banner)
	query.RespondMedia(banner, &telegram.MediaOptions{
		Caption:     fmt.Sprintf(PostText, conf.Title, participants, cat_title, conf.Post),
		ParseMode:   "html",
		ReplyMarkup: kb,
	})
	query.Delete()
}

func admin_conf_edit_title(query *telegram.CallbackQuery, conf *ConfInfo) {
	InsertIntoDB(Db, "UPDATE users SET state = ? WHERE user_id = ?", "admin|edit|"+strconv.FormatInt(conf.ID, 10)+"|title", query.SenderID)
	query.Respond("Введите новое название:", &telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().NewColumn(1,
			telegram.Button.Data("❌ Отмена", "admin|edit|"+strconv.FormatInt(conf.ID, 10)),
		).Build(),
	})
	query.Delete()
}

func admin_conf_edit_banner(query *telegram.CallbackQuery, conf *ConfInfo) {
	InsertIntoDB(Db, "UPDATE users SET state = ? WHERE user_id = ?", "admin|edit|"+strconv.FormatInt(conf.ID, 10)+"|banner", query.SenderID)
	query.Respond("Отправьте новый баннер", &telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().NewColumn(1,
			telegram.Button.Data("❌ Отмена", "admin|edit|"+strconv.FormatInt(conf.ID, 10)),
		).Build(),
	})
	query.Delete()
}

func admin_conf_edit_desc(query *telegram.CallbackQuery, conf *ConfInfo) {
	InsertIntoDB(Db, "UPDATE users SET state = ? WHERE user_id = ?", "admin|edit|"+strconv.FormatInt(conf.ID, 10)+"|desc", query.SenderID)
	query.Respond("Отправьте новое описание", &telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().NewColumn(1,
			telegram.Button.Data("❌ Отмена", "admin|edit|"+strconv.FormatInt(conf.ID, 10)),
		).Build(),
	})
	query.Delete()
}

func admin_conf_edit_cat(query *telegram.CallbackQuery, conf *ConfInfo) {
	args := strings.Split(string(query.Data), "|")
	if len(args) > 4 {
		cat_id, _ := strconv.Atoi(args[4])
		InsertIntoDB(Db, "UPDATE confs SET catalog_id = ? WHERE id = ?", cat_id, conf.ID)

		query.Edit("Категория изменена!", &telegram.SendOptions{
			ReplyMarkup: telegram.NewKeyboard().NewColumn(1,
				telegram.Button.Data("↪️ Обратно", "admin|edit|"+strconv.FormatInt(conf.ID, 10)),
			).Build(),
		})

		return
	}

	cats, err := Db.Query("SELECT * FROM catalogs")
	if err != nil {
		query.Respond("Ошибка: " + err.Error())
		query.Delete()
		return
	}

	var buttons []telegram.KeyboardButton

	for cats.Next() {
		id := 0
		title := ""
		cats.Scan(&id, &title)
		buttons = append(buttons, telegram.Button.Data(title, "admin|edit|"+strconv.FormatInt(conf.ID, 10)+"|cat|"+strconv.Itoa(id)))
	}

	query.Respond("Выберите новую категорию", &telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().
			NewGrid(5, 3, buttons...).
			NewColumn(1,
				telegram.Button.Data("↪️ Обратно", "admin|edit|"+strconv.FormatInt(conf.ID, 10)),
			).Build(),
	})
	query.Delete()
}

func admin_conf_edit_finalize(m *telegram.NewMessage, u *UserInfo) {
	state := strings.Split(u.State, "|")
	id, _ := strconv.ParseInt(state[2], 10, 0)

	so := telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().NewColumn(1,
			telegram.Button.Data("↪️ Обратно", "admin|edit|"+strconv.FormatInt(id, 10)),
		).Build(),
	}

	switch state[3] {
	case "title":
		if m.Text() == "" || m.IsCommand() {
			m.Reply("Недопустимое название. Попробуйте, еще раз", so)
			return
		}
		InsertIntoDB(Db, "UPDATE confs SET title = ? WHERE id = ?", m.Text(), id)
	case "desc":
		if m.Text() == "" || m.IsCommand() {
			m.Reply("Недопустимое описание. Попробуйте, еще раз", so)
			return
		}
		InsertIntoDB(Db, "UPDATE confs SET post = ? WHERE id = ?", m.Text(), id)
	case "banner":
		if m.IsMedia() {
			if p, ok := m.Media().(*telegram.MessageMediaPhoto); ok {
				if _, ok := p.Photo.(*telegram.PhotoObj); ok {
					InsertIntoDB(Db, "UPDATE confs SET banner = ? WHERE id = ?", m.File.FileID, id)
				}
			} else {
				m.Reply("Неправильный формат фото", so)
				return
			}
		} else {
			m.Reply("Неправильный формат фото", so)
			return
		}
	}

	InsertIntoDB(Db, "UPDATE users SET state = ? WHERE user_id = ?", "idle", m.Sender.ID)

	m.Respond("Изменено!", so)
}

func admin_conf_list(query *telegram.CallbackQuery) {
	sql := `
	SELECT confs.id, confs.title
	FROM admins
	JOIN confs ON confs.channel_id=admins.channel_id
	WHERE admin_id = ?`

	cats, err := Db.Query(sql, query.SenderID)
	if err != nil {
		query.Edit("Ошибка")
		return
	}

	var buttons []telegram.KeyboardButton

	for cats.Next() {
		conf_id := 0
		title := ""
		cats.Scan(&conf_id, &title)
		buttons = append(buttons, telegram.Button.Data(title, "admin|edit|"+strconv.Itoa(conf_id)))
	}

	query.Edit("Ваши конфы", &telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().
			NewGrid(3, 5, buttons...).
			NewGrid(1, 1,
				telegram.Button.Data("↪️ Обратно", "admin"),
			).Build(),
	})
}
