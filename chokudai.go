package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Necroforger/dgrouter/exrouter"
	"github.com/bwmarrin/discordgo"
	"github.com/jasonlvhit/gocron"
	_ "github.com/lib/pq"
)

type UserData struct {
	channelID string
	username  string
	atcoderID string
}

var (
	CHOKUDAI_TOKEN  = os.Getenv("DISCORDBOT_CHOKUDAI_TOKEN")
	CHOKUDAI_PREFIX = "/chokudai "

	CHOKUDAI_ALERT_TIME = "20:00"

	CHOKUDAI_MESSAGE_BIBLE         = "だから慶應は学歴自慢じゃないっつーの。慶應という学歴が俺を高めるんじゃない。俺という存在が慶應という学歴の価値を高めるんだよ。"
	CHOKUDAI_MESSAGE_INTERNALERROR = "内部エラーが発生しました。"
)

var chokudaiDiscord *discordgo.Session
var chokudaiDB *sql.DB

func initChokudai() error {
	log.Println("Initializing Chokudai...")
	var err error

	chokudaiDB, err = sql.Open("postgres", DATABASE_URL)
	if err != nil {
		return err
	}

	chokudaiDiscord, err = discordgo.New("Bot " + CHOKUDAI_TOKEN)
	if err != nil {
		return err
	}

	router := exrouter.New()

	router.On("register", func(ctx *exrouter.Context) {
		var err error
		if len(ctx.Args) != 2 {
			ctx.Reply("AtCoder IDを１つ指定して下さい。")
			return
		}
		atcoderID := ctx.Args[1]
		err = registerChokudaiUser(ctx.Msg.GuildID, ctx.Msg.ChannelID, ctx.Msg.Author.Username, atcoderID)
		if err != nil {
			log.Println(err)
			ctx.Reply(CHOKUDAI_MESSAGE_INTERNALERROR)
			return
		}
		ctx.Reply(atcoderID + "を登録しました。")
	}).Desc("AtCoder IDを登録します。")

	router.On("deregister", func(ctx *exrouter.Context) {
		var err error
		err = deregisterChokudaiUser(ctx.Msg.GuildID, ctx.Msg.Author.Username)
		if err != nil {
			log.Println(err)
			ctx.Reply(CHOKUDAI_MESSAGE_INTERNALERROR)
			return
		}
		ctx.Reply("登録を解除しました。")
	}).Desc("AtCoder IDの登録を解除します。")

	router.On("bible", func(ctx *exrouter.Context) {
		ctx.Reply(CHOKUDAI_MESSAGE_BIBLE)
	}).Desc("バイブル")

	router.Default = router.On("help", func(ctx *exrouter.Context) {
		var text = "このBotは競技プログラミングに関するいろいろなことを扱います。\nソース：https://github.com/makutamoto/discordbots\n"
		text += "```"
		for _, v := range router.Routes {
			text += fmt.Sprintf(CHOKUDAI_PREFIX+"%-10s: %s\n", v.Name, v.Description)
		}
		text += "```"
		ctx.Reply(text)
	}).Desc("このメッセージを表示します。")

	chokudaiDiscord.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == chokudaiDiscord.State.User.ID {
			return
		}
		router.FindAndExecute(chokudaiDiscord, CHOKUDAI_PREFIX, chokudaiDiscord.State.User.ID, m.Message)
	})
	chokudaiDiscord.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)
	log.Println("Opening Chokudai...")
	err = chokudaiDiscord.Open()
	if err != nil {
		return err
	}

	gocron.Every(1).Day().At(CHOKUDAI_ALERT_TIME).Do(func() {
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		err := chokudaiAlert(today.Unix())
		if err != nil {
			log.Println(err)
		}
	})

	log.Println("Done")
	return nil
}

func deinitChokudai() {
	log.Println("Deinitializing Chokudai...")
	var err error
	err = chokudaiDiscord.Close()
	if err != nil {
		log.Println(err)
		return
	}
	err = chokudaiDB.Close()
	if err != nil {
		log.Println(err)
	}
	log.Println("Done")
}

func registerChokudaiUser(guildID string, channelID string, username string, atcoderID string) error {
	var err error
	_, err = chokudaiDB.Exec(`
		INSERT INTO chokudai_users
		(guild_id, username, channel_id, atcoder_id)
		VALUES ($1, $2, $3, $4) ON CONFLICT(guild_id, username)
		DO UPDATE SET channel_id=$3, atcoder_id=$4;
	`, guildID, username, channelID, atcoderID)
	return err
}

func deregisterChokudaiUser(guildID string, username string) error {
	var err error
	_, err = chokudaiDB.Exec(`
		DELETE FROM chokudai_users
		WHERE guild_id = $1 AND username = $2;
	`, guildID, username)
	return err
}

func getRegisteredChokudaiUsers() ([]UserData, error) {
	var err error
	var users []UserData
	rows, err := chokudaiDB.Query(`SELECT username, channel_id, atcoder_id FROM chokudai_users;`)
	if err != nil {
		return users, err
	}
	for rows.Next() {
		var user UserData
		err = rows.Scan(&user.username, &user.channelID, &user.atcoderID)
		if err != nil {
			return users, err
		}
		users = append(users, user)
	}
	return users, nil
}

func chokudaiSay(channelID string, text string) error {
	_, err := chokudaiDiscord.ChannelMessageSend(channelID, text)
	return err
}

func chokudaiAlert(date int64) error {
	var err error
	users, err := getRegisteredChokudaiUsers()
	if err != nil {
		return err
	}
	for _, user := range users {
		submissions, err := getAtCoderUserSubmissions(user.atcoderID)
		if err != nil {
			log.Println(err)
		}
		filteredSubmissions := filterAtCoderSubmissionsByUniqueAC(submissions)
		filteredSubmissions = filterAtCoderSubmissionsByDate(filteredSubmissions, date)
		if len(filteredSubmissions) == 0 {
			message := fmt.Sprintf("@%s 競プロしろ(AtCoder ID: %s)", user.username, user.atcoderID)
			err = chokudaiSay(user.channelID, message)
			if err != nil {
				log.Println(err)
			}
		}
	}
	return nil
}
