package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/amarnathcjd/gogram/telegram"
)

func conf_handler(client *telegram.Client) {
	client.On("callback:conf", func(query *telegram.CallbackQuery) error {
		args := strings.Split(string(query.Data), "|")
		confid, _ := strconv.ParseInt(args[1], 10, 0)

		conf := new(ConfInfo)

		res := Db.QueryRow("SELECT * FROM confs WHERE id = ?", confid)
		err := res.Scan(&conf.ID, &conf.Title, &conf.Post, &conf.Banner, &conf.ChannelID, &conf.CatalogID)
		if err != nil {
			query.Edit("Ошибка: " + err.Error())
			return nil
		}

		if len(args) > 2 {
			switch args[2] {
			case "like":
				conf_like(query, conf)
			case "dislike":
				conf_dislike(query, conf)
			case "enter":
				conf_enter(client, query, conf)
				return nil
			}
		}

		conf_show(client, query, conf)

		return nil
	})
}

func conf_show(client *telegram.Client, query *telegram.CallbackQuery, conf *ConfInfo) {
	var cat_title string
	res := Db.QueryRow("SELECT name FROM catalogs WHERE id = ?", conf.CatalogID)
	err := res.Scan(&cat_title)
	if err != nil {
		query.Edit("Ошибка: " + err.Error())
		return
	}

	channel, _ := client.GetChannel(conf.ChannelID)

	text := fmt.Sprintf(PostText, conf.Title, channel.ParticipantsCount, cat_title, conf.Post)
	kb := telegram.NewKeyboard().NewGrid(1, 2,
		telegram.Button.Data("👍 "+strconv.Itoa(get_count("likes", conf)), "conf|"+strconv.FormatInt(conf.ID, 10)+"|like"),
		telegram.Button.Data("👎 "+strconv.Itoa(get_count("dislikes", conf)), "conf|"+strconv.FormatInt(conf.ID, 10)+"|dislike"),
	).NewGrid(2, 1,
		telegram.Button.Data("Войти", "conf|"+strconv.FormatInt(conf.ID, 10)+"|enter"),
		telegram.Button.Data("↪️ Обратно", "search"),
	).Build()

	if conf.Banner == "" {
		query.Respond(text, &telegram.SendOptions{
			ParseMode:   "html",
			ReplyMarkup: kb,
		})
		log.Println(err)
	} else {
		banner, _ := telegram.ResolveBotFileID(conf.Banner)
		query.RespondMedia(banner, &telegram.MediaOptions{
			Caption:     text,
			ParseMode:   "html",
			ReplyMarkup: kb,
		})
	}
	query.Delete()
}

func get_count(what string, conf *ConfInfo) int {
	var count int
	res := Db.QueryRow("SELECT count(*) FROM "+what+" WHERE conf_id = ?", conf.ID)
	err := res.Scan(&count)
	if err != nil {
		return count
	}
	return count
}

func has_emote(what string, conf *ConfInfo, user_id int64) bool {
	var count int
	res := Db.QueryRow("SELECT count(*) FROM "+what+" WHERE conf_id = ? AND user_id = ?", conf.ID, user_id)
	err := res.Scan(&count)
	if err != nil {
		return false
	}
	return count != 0
}

func conf_like(query *telegram.CallbackQuery, conf *ConfInfo) {
	if has_emote("like", conf, query.SenderID) {
		InsertIntoDB(Db, "DELETE FROM likes WHERE user_id = ? AND conf_id = ?", query.SenderID, conf.ID)
	} else {
		InsertIntoDB(Db, "INSERT INTO likes (user_id, conf_id) VALUES (?, ?)", query.SenderID, conf.ID)
		InsertIntoDB(Db, "DELETE FROM dislikes WHERE user_id = ? AND conf_id = ?", query.SenderID, conf.ID)
	}
}

func conf_dislike(query *telegram.CallbackQuery, conf *ConfInfo) {
	if has_emote("dislike", conf, query.SenderID) {
		InsertIntoDB(Db, "DELETE FROM dislikes WHERE user_id = ? AND conf_id = ?", query.SenderID, conf.ID)
	} else {
		InsertIntoDB(Db, "INSERT INTO dislikes (user_id, conf_id) VALUES (?, ?)", query.SenderID, conf.ID)
		InsertIntoDB(Db, "DELETE FROM likes WHERE user_id = ? AND conf_id = ?", query.SenderID, conf.ID)
	}
}

func conf_enter(client *telegram.Client, query *telegram.CallbackQuery, conf *ConfInfo) {
	invite, err := client.GetChatInviteLink(conf.ChannelID, &telegram.InviteLinkOptions{
		Limit: 1,
	})
	if err != nil {
		log.Panicln(err)
		os.Exit(1)
	}

	query.Respond("Ссылка: "+invite.(*telegram.ChatInviteExported).Link, &telegram.SendOptions{
		ReplyMarkup: telegram.NewKeyboard().NewGrid(1, 1,
			telegram.Button.Data("↪️ Обратно", "conf|"+strconv.FormatInt(conf.ID, 10)),
		).Build(),
	})
	query.Delete()
}
