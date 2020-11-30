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

const startGamePattern = "start <@!(?P<target>\\d{18})> ?$"
const getGamePattern = "get <@!(?P<target>\\d{18})> ?$"
const getMovesPattern = "get moves <@!(?P<target>\\d{18})> ?$"
const movePattern = "move <@!(?P<target>\\d{18})> ([A|B|C|D|E|F|G|H][8|7|6|5|4|3|2|1]) ([A|B|C|D|E|F|G|H][8|7|6|5|4|3|2|1]) ?$"
const promotionPattern = "move promotion <@!(?P<target>\\d{18})> ([A|B|C|D|E|F|G|H][8|7|6|5|4|3|2|1]) ([A|B|C|D|E|F|G|H][8|7|6|5|4|3|2|1]) (rook|knight|queen|bishop) ?$"
const castlingPattern = "castling <@!(?P<target>\\d{18})> ([A|B|C|D|E|F|G|H][8|7|6|5|4|3|2|1]) ([A|B|C|D|E|F|G|H][8|7|6|5|4|3|2|1]) ([A|B|C|D|E|F|G|H][8|7|6|5|4|3|2|1]) ([A|B|C|D|E|F|G|H][8|7|6|5|4|3|2|1]) ?$"
const enPassantPattern = "en passant <@!(?P<target>\\d{18})> ([A|B|C|D|E|F|G|H][8|7|6|5|4|3|2|1]) ?$"
const resginPattern = "resign <@!(?P<target>\\d{18})> ?$"
const codeInfoPattern = "code info$"
const infoPattern = "info$"

var (
	commandSet  *discom.CommandSet
	startGameRe = regexp.MustCompile(startGamePattern)
	getGameRe   = regexp.MustCompile(getGamePattern)
	moveRe      = regexp.MustCompile(movePattern)
	resginRe    = regexp.MustCompile(resginPattern)
	castlingRe  = regexp.MustCompile(castlingPattern)
	enPassantRe = regexp.MustCompile(enPassantPattern)
	getMovesRe  = regexp.MustCompile(getMovesPattern)
	promotionRe = regexp.MustCompile(promotionPattern)
	dbIns       *db.Instance
)

func init() {
	commandSet = discom.CreateCommandSet(false, regexp.MustCompile("cb"))

	err := commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(infoPattern), Handler: infoCmd,
		Description: "prints more info about how the bot works",
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
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
		Description: "get game a target game",
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(getMovesPattern), Handler: getMovesCmd,
		Description: "prints a move list and creates a gif of all moves so far",
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(movePattern), Handler: moveCmd,
		Description: "move a piece in a target game it goes `cb move @player FROM TO`",
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(castlingPattern), Handler: castlingCmd,
		Description: "perform a castling action it goes `cb move @player FROM TO FROM TO` REMEMBER THIS HAS NO RULE CHECKING",
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(enPassantPattern), Handler: enPassantCmd,
		Description: "perform a En Passant action it goes `cb en passant @player TARGET` this will remove the piece that was En Passanted",
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

func infoCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(
		m.ChannelID,
		fmt.Sprintf(
			"<@!%s>: Here is some info\n"+
				"* When you see comands and this `<@!(?P<target>\\d{18})>` you should enter @ somebody\n"+
				"* There is NO rule checking it's up to you and your opponent to not be shit cunts once somebody is in check mate they should resgin\n"+
				"* To Castle you need to use a seprate move command see help for more info\n"+
				"* To En Passant you need to use a seprate command after moving see help for more info", m.Author.ID,
		),
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

	_, err := getGame(m, target)
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

func printMissingGame(s *discordgo.Session, m *discordgo.MessageCreate) {
	s.ChannelMessageSend(
		m.ChannelID,
		fmt.Sprintf("<@!%s>: error getting game, game doesn't exist", m.Author.ID),
	)
}

func getGame(m *discordgo.MessageCreate, target string) (*chess.Game, error) {
	return dbIns.GetGame(m.Author.ID, target, m.GuildID)
}

func getGameCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	matches := getGameRe.FindAllStringSubmatch(strings.ToLower(m.Content), -1)

	target := matches[0][1]

	game, err := getGame(m, target)
	if err != nil {
		printMissingGame(s, m)
		return
	}

	msg := fmt.Sprintf(
		"Match between <@!%s>: %s and <@!%s>: %s",
		game.White.ID, game.White.Side.String(), game.Black.ID, game.Black.Side.String(),
	)
	sendGame(s, m.ChannelID, msg, game)
}

func getMovesCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	matches := getMovesRe.FindAllStringSubmatch(strings.ToLower(m.Content), -1)

	target := matches[0][1]

	game, err := getGame(m, target)
	if err != nil {
		printMissingGame(s, m)
		return
	}

	msg := fmt.Sprintf(
		"Match between <@!%s>: %s and <@!%s>: %s all moves:\n%v",
		game.White.ID, game.White.Side.String(), game.Black.ID,
		game.Black.Side.String(), game.MovesAtomicNotation(),
	)
	s.ChannelMessageSendComplex(
		m.ChannelID,
		&discordgo.MessageSend{
			Content: msg,
			Files: []*discordgo.File{&discordgo.File{
				Name: fmt.Sprintf("%s.gif", game.ID()), ContentType: "gif",
				Reader: game.CreateGif(),
			}},
		},
	)
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

	game, err := getGame(m, target)
	if err != nil {
		printMissingGame(s, m)
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

func enPassantCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	matches := enPassantRe.FindAllStringSubmatch(m.Content, -1)

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

	playerTarget := matches[0][1]
	pieceTarget := matches[0][2]

	game, err := getGame(m, playerTarget)
	if err != nil {
		printMissingGame(s, m)
		return
	}

	game.MakeMove(chess.Move{
		From: game.FindEmptySqaure(),
		To:   chess.StringToPostion(pieceTarget),
	})

	game.Turn = game.GetOpponent(m.Author.ID).Side

	dbIns.SaveGame(game)

	msg := fmt.Sprintf(
		"Match between <@!%s>: %s and <@!%s>: %s En Passant",
		game.White.ID, game.White.Side.String(), game.Black.ID, game.Black.Side.String(),
	)
	sendGame(s, m.ChannelID, msg, game)
}

func castlingCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	matches := castlingRe.FindAllStringSubmatch(m.Content, -1)

	if matches == nil {
		s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"<@!%s> Invalid castling move try using uppercase",
				m.Author.ID,
			),
		)
		return
	}

	target := matches[0][1]
	aFrom := matches[0][2]
	aTo := matches[0][3]

	bFrom := matches[0][4]
	bTo := matches[0][5]

	game, err := getGame(m, target)
	if err != nil {
		printMissingGame(s, m)
		return
	}

	if game.Turn != game.GetPlayer(m.Author.ID).Side {
		s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"<@!%s> Invalid castling move it's not your turn",
				m.Author.ID,
			),
		)
		return
	}

	game.MakeMove(chess.Move{
		From: chess.StringToPostion(aFrom),
		To:   chess.StringToPostion(aTo),
	})
	game.MakeMove(chess.Move{
		From: chess.StringToPostion(bFrom),
		To:   chess.StringToPostion(bTo),
	})

	game.Turn = game.GetOpponent(m.Author.ID).Side

	dbIns.SaveGame(game)

	msg := fmt.Sprintf(
		"Match between <@!%s>: %s and <@!%s>: %s castling move",
		game.White.ID, game.White.Side.String(), game.Black.ID, game.Black.Side.String(),
	)
	sendGame(s, m.ChannelID, msg, game)
}

func resginCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	matches := resginRe.FindAllStringSubmatch(strings.ToLower(m.Content), -1)

	target := matches[0][1]

	game, err := getGame(m, target)
	if err != nil {
		printMissingGame(s, m)
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
	s.ChannelMessageSendComplex(
		m.ChannelID,
		&discordgo.MessageSend{
			Content: msg,
			Files: []*discordgo.File{&discordgo.File{
				Name: fmt.Sprintf("%s.gif", game.ID()), ContentType: "gif",
				Reader: game.CreateGif(),
			}},
		},
	)
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
