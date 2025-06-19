package main

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func search_handlers(client *telegram.Client) {
	client.On("callback:search", func(query *telegram.CallbackQuery) error {
		args := strings.Split(string(query.Data), "|")
		if len(args) > 1 {
			switch args[1] {
			case "text":
				search_by_text(query)
				return nil
			case "cat":
				return search_by_cats(query)
			case "random":
				return search_random(client, query)
			}
		}

		InsertIntoDB(Db, "UPDATE users SET state = ? WHERE user_id = ?", "idle", query.SenderID)

		query.Respond("📥 Поищем конфы?", &telegram.SendOptions{
			ReplyMarkup: telegram.NewKeyboard().NewGrid(3, 1,
				telegram.Button.Data("👑 Мне повезет!", "search|random"),
				telegram.Button.Data("💿 Поиск по названию", "search|text"),
				telegram.Button.Data("💼 Поиск по категориям", "search|cat"),
			).NewGrid(1, 1,
				telegram.Button.Data("↪️ Обратно", "start"),
			).Build(),
		})
		query.Delete()
		return nil
	})
}

func search_random(client *telegram.Client, query *telegram.CallbackQuery) error {
	var count int
	res := Db.QueryRow("SELECT count(*) FROM confs")
	res.Scan(&count)
	if count == 0 {
		query.Edit("Конф нет, либо произошла внутренняя ошибка", &telegram.SendOptions{
			ReplyMarkup: telegram.NewKeyboard().
				NewGrid(1, 1,
					telegram.Button.Data("↪️ Обратно", "search"),
				).Build(),
		})
		return nil
	}

	conf := new(ConfInfo)

	res = Db.QueryRow("SELECT * FROM confs LIMIT 1 OFFSET ?", rand.Int63n(int64(count)))
	err := res.Scan(&conf.ID, &conf.Title, &conf.Post, &conf.Banner, &conf.ChannelID, &conf.CatalogID)
	if err != nil {
		query.Edit("Ошибка: " + err.Error())
		return nil
	}

	conf_show(client, query, conf)

	return nil
}

func search_by_cats(query *telegram.CallbackQuery) error {
	args := strings.Split(string(query.Data), "|")
	if len(args) > 2 {
		catid, _ := strconv.Atoi(args[2])
		pid, _ := strconv.Atoi(args[3])

		count := 0
		res := Db.QueryRow("SELECT count(*) FROM confs WHERE catalog_id = ?", catid)
		res.Scan(&count)

		limit_pid := int(math.Ceil(float64(count)/float64(PostLimit))) - 1
		var paginator []telegram.KeyboardButton
		if limit_pid > 0 {
			if pid > 0 {
				paginator = append(paginator, telegram.Button.Data("◀️", "search|cat|"+args[2]+"|"+strconv.Itoa(pid-1)))
			}

			if pid < limit_pid {
				paginator = append(paginator, telegram.Button.Data("▶️", "search|cat|"+args[2]+"|"+strconv.Itoa(pid+1)))
			}
		}

		cats, err := Db.Query("SELECT id, title FROM confs WHERE catalog_id = ? LIMIT ? OFFSET ?", catid, PostLimit, PostLimit*pid)
		if err != nil {
			query.Edit("Ошибка")
			return err
		}

		var buttons []telegram.KeyboardButton

		for cats.Next() {
			conf_id := 0
			title := ""
			cats.Scan(&conf_id, &title)
			buttons = append(buttons, telegram.Button.Data(title, "conf|"+strconv.Itoa(conf_id)))
		}

		query.Edit("📌 Выберите конфу", &telegram.SendOptions{
			ReplyMarkup: telegram.NewKeyboard().
				NewGrid(5, 1, buttons...).
				NewGrid(2, 1, paginator...).
				NewGrid(1, 1,
					telegram.Button.Data("↪️ Обратно", "search"),
				).Build(),
		})

		return nil
	}

	cats, err := Db.Query(`
		SELECT id, name, (
			SELECT COUNT(*) FROM confs
			WHERE confs.catalog_id = catalogs.id
		) AS c FROM catalogs
	`)
	if err != nil {
		query.Edit("Ошибка")
		return err
	}

	var buttons []telegram.KeyboardButton

	for cats.Next() {
		id := 0
		title := ""
		count := 0

		cats.Scan(&id, &title, &count)
		buttons = append(buttons, telegram.Button.Data(fmt.Sprintf("%s (%v)", title, count), "search|cat|"+strconv.Itoa(id)+"|0"))
	}

	query.Edit("📚 Категории", &telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().
			NewGrid(10, 2, buttons...).
			NewGrid(1, 1,
				telegram.Button.Data("↪️ Обратно", "search"),
			).Build(),
	})

	return nil
}

func search_by_text(query *telegram.CallbackQuery) {
	args := strings.Split(string(query.Data), "|")
	if len(args) > 2 {
		pid, _ := strconv.Atoi(args[3])
		query.Edit("📌 Выберите конфу", &telegram.SendOptions{
			ReplyMarkup: get_by_text(args[2], pid),
		})
		return
	}

	InsertIntoDB(Db, "UPDATE users SET state = ? WHERE user_id = ?", "search|text", query.SenderID)

	query.Edit("Введите название по которому будем искать", &telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().NewGrid(1, 1,
			telegram.Button.Data("↪️ Обратно", "search"),
		).Build(),
	})
}

func get_by_text(text string, pid int) *telegram.ReplyInlineMarkup {
	que := "%" + text + "%"

	count := 0
	res := Db.QueryRow("SELECT count(*) FROM confs WHERE title LIKE ? OR post LIKE ?", que, que)
	res.Scan(&count)

	limit_pid := int(math.Ceil(float64(count) / float64(PostLimit)))
	var paginator []telegram.KeyboardButton
	if limit_pid > 0 {
		if pid > 0 {
			paginator = append(paginator, telegram.Button.Data("◀️", "search|text|"+text+"|"+strconv.Itoa(pid-1)))
		}

		if pid < limit_pid {
			paginator = append(paginator, telegram.Button.Data("▶️", "search|text|"+text+"|"+strconv.Itoa(pid+1)))
		}
	}

	cats, err := Db.Query("SELECT id, title FROM confs WHERE title LIKE ? OR post LIKE ? LIMIT ? OFFSET ?", que, que, PostLimit, PostLimit*pid)
	if err != nil {
		return nil
	}

	var buttons []telegram.KeyboardButton

	for cats.Next() {
		conf_id := 0
		title := ""
		cats.Scan(&conf_id, &title)
		buttons = append(buttons, telegram.Button.Data(title, "conf|"+strconv.Itoa(conf_id)))
	}

	return telegram.NewKeyboard().
		NewGrid(5, 1, buttons...).
		NewGrid(2, 1, paginator...).
		NewGrid(1, 1,
			telegram.Button.Data("↪️ Обратно", "search"),
		).Build()
}

func search_by_text_final(m *telegram.NewMessage) error {
	InsertIntoDB(Db, "UPDATE users SET state = ? WHERE user_id = ?", "idle", m.Sender.ID)

	m.Respond("📌 Выберите конфу", telegram.SendOptions{
		ReplyMarkup: get_by_text(m.Text(), 0),
	})

	return nil
}
