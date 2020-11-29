package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/sardap/chessbot/chess"
	"github.com/sardap/chessbot/db"
	"github.com/sardap/discom"
)

const startGamePattern = "start.*?<@!(?P<target>\\d{18})>$"
const getGamePattern = "get.*?<@!(?P<target>\\d{18})>$"
const movePattern = "move.*?<@!(?P<target>\\d{18})>*.?([A|B|C|D|E|F|G|H][8|7|6|5|4|3|2|1]) ([A|B|C|D|E|F|G|H][8|7|6|5|4|3|2|1])$"
const resginPattern = "resign.*?<@!(?P<target>\\d{18})>$"
const codeInfoPattern = "code.*?info$"

var (
	commandSet  *discom.CommandSet
	startGameRe = regexp.MustCompile(startGamePattern)
	getGameRe   = regexp.MustCompile(getGamePattern)
	moveRe      = regexp.MustCompile(movePattern)
	resginRe    = regexp.MustCompile(resginPattern)
	dbIns       *db.Instance
)

func init() {
	commandSet = discom.CreateCommandSet(false, regexp.MustCompile("cb"))

	err := commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(codeInfoPattern), Handler: codeInfoCmd,
		Description: "prints the code info",
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(startGamePattern), Handler: startGameCmd,
		Description: "start game with the target player (you can only have a single game going with a player per server)",
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(getGamePattern), Handler: getGameCmd,
		Description: "get game",
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(movePattern), Handler: moveCmd,
		Description: "move a piece in a target game",
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(resginPattern), Handler: resginCmd,
		Description: "resign from a target game",
	})
	if err != nil {
		panic(err)
	}
}

func sendGame(s *discordgo.Session, channelID, msg string, game *chess.Game) {
	s.ChannelMessageSendComplex(
		channelID,
		&discordgo.MessageSend{
			Content: msg,
			Files: []*discordgo.File{&discordgo.File{
				Name: fmt.Sprintf("%s.jpeg", game.ID()), ContentType: "jpeg",
				Reader: game.CreateImage(),
			}},
		},
	)
}

func codeInfoCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(
		m.ChannelID,
		fmt.Sprintf(
			"<@!%s>: You can go here to see the source code and make contributions here https://github.com/sardap/chessbot", m.Author.ID,
		),
	)
}

func startGameCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	matches := startGameRe.FindAllStringSubmatch(strings.ToLower(m.Content), -1)

	target := matches[0][1]

	if target == m.Author.ID {
		s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"<@!%s>: You cannot play with yourself god is watching", m.Author.ID,
			),
		)
		return
	}

	_, err := getGame(s, m, target)
	if err == nil {
		s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"<@!%s>: You already have a game going with that player", m.Author.ID,
			),
		)
		return

	}

	game := chess.CreateGame(m.Author.ID, target, m.GuildID)
	dbIns.SaveGame(&game)

	msg := fmt.Sprintf(
		"New Match Between <@!%s>: %s and <@!%s>: %s",
		game.White.ID, game.White.Side.String(), game.Black.ID, game.Black.Side.String(),
	)
	sendGame(s, m.ChannelID, msg, &game)
}

func getGame(s *discordgo.Session, m *discordgo.MessageCreate, target string) (*chess.Game, error) {
	game, err := dbIns.GetGame(m.Author.ID, target, m.GuildID)
	if err != nil {
		s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf("<@!%s>: error getting game! %v", m.Author.ID, err),
		)
		return nil, err
	}

	return game, nil
}

func getGameCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	matches := getGameRe.FindAllStringSubmatch(strings.ToLower(m.Content), -1)

	target := matches[0][1]

	game, err := getGame(s, m, target)
	if err != nil {
		return
	}

	msg := fmt.Sprintf(
		"Match between <@!%s>: %s and <@!%s>: %s",
		game.White.ID, game.White.Side.String(), game.Black.ID, game.Black.Side.String(),
	)
	sendGame(s, m.ChannelID, msg, game)
}

func moveCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	matches := moveRe.FindAllStringSubmatch(m.Content, -1)

	if matches == nil {
		s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"<@!%s> Invalid Move try using uppercase",
				m.Author.ID,
			),
		)
		return
	}

	target := matches[0][1]
	from := matches[0][2]
	to := matches[0][3]

	game, err := getGame(s, m, target)
	if err != nil {
		return
	}

	mv := chess.Move{
		From: chess.StringToPostion(from),
		To:   chess.StringToPostion(to),
	}

	if !game.ValidMove(m.Author.ID, mv) {
		s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"<@!%s> Invalid Move",
				m.Author.ID,
			),
		)
		return
	}

	game.MakeMove(mv)

	dbIns.SaveGame(game)

	msg := fmt.Sprintf(
		"Match between <@!%s>: %s and <@!%s>: %s Move %s to %s",
		game.White.ID, game.White.Side.String(), game.Black.ID, game.Black.Side.String(), from, to,
	)
	sendGame(s, m.ChannelID, msg, game)
}

func resginCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	matches := resginRe.FindAllStringSubmatch(strings.ToLower(m.Content), -1)

	target := matches[0][1]

	game, err := getGame(s, m, target)
	if err != nil {
		return
	}

	game.Winner = game.GetOpponent(m.Author.ID).Side

	err = dbIns.DeleteGame(game)
	if err != nil {
		s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf("<@!%s>: error deleting game! %v", m.Author.ID, err),
		)
	}
	go dbIns.ArchiveGame(game)

	msg := fmt.Sprintf(
		"Match between <@!%s>: %s and <@!%s>: %s Final State\n"+
			"🎉Winner🎉 <@!%s>",
		game.White.ID, game.White.Side.String(), game.Black.ID, game.Black.Side.String(),
		game.GetOpponent(m.Author.ID).ID,
	)
	sendGame(s, m.ChannelID, msg, game)
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
}

func main() {
	fmt.Printf("Connecting to DB")
	dbIns = &db.Instance{}
	dbIns.Connect()

	token := strings.Replace(os.Getenv("DISCORD_AUTH"), "\"", "", -1)
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Printf("unable to create new discord instance")
		log.Fatal(err)
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	discord.AddHandler(commandSet.Handler)
	discord.AddHandler(messageCreate)

	// Open a websocket connection to Discord and begin listening.
	err = discord.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	discord.UpdateStatus(-1, "\"cb help\"")

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	discord.Close()

}
