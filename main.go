package main

import (
	"errors"
	"fmt"
	"image/color"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/icza/gox/imagex/colorx"
	"github.com/sardap/chessbot/chess"
	"github.com/sardap/chessbot/db"
	"github.com/sardap/chessbot/env"
	"github.com/sardap/discom"
)

const infoPattern = "info$"
const codeInfoPattern = "code info$"
const startGamePattern = "<@!(?P<target>\\d{18})> .*?start ?((?P<white_color>#[0-9a-f]{6}) ?(?P<black_color>#[0-9a-f]{6}))? ?$"
const getGamePattern = "<@!(?P<target>\\d{18})> .*?get$"
const getMovesPattern = "<@!(?P<target>\\d{18})> .*?get .*?moves?$"
const movePattern = "<@!(?P<target>\\d{18})> .*?move .*?([a-h][1-8]) .*?([a-h][1-8]) ?$"
const castlingPattern = "<@!(?P<target>\\d{18})> .*?castling .*?([a-h][1-8]) .*?([a-h][1-8]) ([a-h][1-8]) ([a-h][1-8]) ?$"
const enPassantPattern = "<@!(?P<target>\\d{18})> .*?en .*?passant .*?([a-h][1-8]) ?$"
const promotionPattern = "<@!(?P<target>\\d{18})> .*?move .*?promotion .*?([a-h][1-8]) .*?([a-h][1-8]) (rook|knight|queen|bishop) ?$"
const resginPattern = "<@!(?P<target>\\d{18})> .*?(resign|resgin)$"

var (
	commandSet  *discom.CommandSet
	startGameRe = regexp.MustCompile(startGamePattern)
	getGameRe   = regexp.MustCompile(getGamePattern)
	getMovesRe  = regexp.MustCompile(getMovesPattern)
	moveRe      = regexp.MustCompile(movePattern)
	castlingRe  = regexp.MustCompile(castlingPattern)
	enPassantRe = regexp.MustCompile(enPassantPattern)
	promotionRe = regexp.MustCompile(promotionPattern)
	resginRe    = regexp.MustCompile(resginPattern)
	dbIns       *db.Instance
)

func init() {
	commandSet = discom.CreateCommandSet(regexp.MustCompile(env.CmdPrefix))

	err := commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(infoPattern), Handler: infoCmd,
		Example:     "info",
		Description: "Prints more info about how the bot works",
		CaseInSense: true,
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(codeInfoPattern), Handler: codeInfoCmd,
		Example: "code info", Description: "Prints the code info",
		CaseInSense: true,
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(startGamePattern), Handler: startGameCmd,
		Example:     "@TARGET_PLAYER start",
		Description: "Start game with the target player (you can only have a single game going with a player per server)",
		CaseInSense: true,
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(getGamePattern), Handler: getGameCmd,
		Example:     "@TARGET_PLAYER get",
		Description: "View curent state of board for game",
		CaseInSense: true,
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(getMovesPattern), Handler: getMovesCmd,
		Example:     "@TARGET_PLAYER get moves",
		Description: "Prints a move list and creates a gif of all moves so far",
		CaseInSense: true,
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(movePattern), Handler: moveCmd,
		Example:     "@TARGET_PLAYER move F2 F4",
		Description: "Move a piece in a target game it uses the letters and numbers grid thing",
		CaseInSense: true,
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(castlingPattern), Handler: castlingCmd,
		Example:     "@TARGET_PLAYER castling A1 D1 D1 A1",
		Description: "Perform a castling action it goes REMEMBER THIS HAS NO RULE CHECKING",
		CaseInSense: true,
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(enPassantPattern), Handler: enPassantCmd,
		Example:     "@TARGET_PLAYER en passant A1 A2",
		Description: "Perform a En Passant action it goes this will remove the piece that was En Passanted NOT MOVE it",
		CaseInSense: true,
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(promotionPattern), Handler: movePromotionCmd,
		Example:     "@TARGET_PLAYER move promotion A1 A2 rook",
		Description: "Perform a pawn promotion valid values are rook, knight, queen, bishop",
		CaseInSense: true,
	})
	if err != nil {
		panic(err)
	}

	err = commandSet.AddCommand(discom.Command{
		Re: regexp.MustCompile(resginPattern), Handler: resginCmd,
		Example: "@TARGET_PLAYER resgin", Description: "Resign from a target game",
		CaseInSense: true,
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
			Files: []*discordgo.File{{
				Name:   fmt.Sprintf("%s.png", game.ID()),
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

	var white, black string
	if rand.Float32() > 0.5 {
		white = m.Author.ID
		black = target
	} else {
		black = m.Author.ID
		white = target
	}

	var whiteColor, blackColor color.RGBA
	if matches[0][3] != "" {
		whiteColor, _ = colorx.ParseHexColor(matches[0][3])
		blackColor, _ = colorx.ParseHexColor(matches[0][4])
	} else {
		whiteColor = color.RGBA{255, 255, 255, 255}
		blackColor = color.RGBA{0, 0, 0, 255}
	}

	game := chess.CreateGame(white, black, m.GuildID, whiteColor, blackColor)
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
	return dbIns.GetGame(chess.GameID(m.GuildID, m.Author.ID, target))
}

func rgbaToString(color color.RGBA) string {
	return fmt.Sprintf("#%x%x%x", color.R, color.G, color.G)
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
		"Match between <@!%s>: %s and <@!%s>: %s\n"+
			"White Color: %s Black Color: %s",
		game.White.ID, game.White.Side.String(), game.Black.ID, game.Black.Side.String(),
		rgbaToString(game.White.Color), rgbaToString(game.Black.Color),
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
		game.Black.Side.String(), game.AlgebraicNotation(),
	)
	s.ChannelMessageSendComplex(
		m.ChannelID,
		&discordgo.MessageSend{
			Content: msg,
			Files: []*discordgo.File{{
				Name: fmt.Sprintf("%s.gif", game.ID()), ContentType: "gif",
				Reader: game.CreateGif(),
			}},
		},
	)
}

func moveCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	matches := moveRe.FindAllStringSubmatch(strings.ToLower(m.Content), -1)

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

	if err := game.ValidMove(m.Author.ID, mv); err != nil {
		if errors.Is(err, chess.ErrCheckMate) {
			var side chess.SideType
			//This is shit
			if strings.Contains(strings.ToLower(err.Error()), "white") {
				side = chess.SideWhite
			} else {
				side = chess.SideBlack
			}
			resgin(s, m, target, side)
			return
		}

		s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"<@!%s> Invalid Move %s",
				m.Author.ID, err,
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

func castlingCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	matches := castlingRe.FindAllStringSubmatch(strings.ToLower(m.Content), -1)

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

func enPassantCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	matches := enPassantRe.FindAllStringSubmatch(strings.ToLower(m.Content), -1)

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

func movePromotionCmd(s *discordgo.Session, m *discordgo.MessageCreate) {
	matches := promotionRe.FindAllStringSubmatch(strings.ToLower(m.Content), -1)

	if matches == nil {
		s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"<@!%s> invalid promo move",
				m.Author.ID,
			),
		)
		return
	}

	target := matches[0][1]
	from := matches[0][2]
	to := matches[0][3]
	promo := matches[0][4]

	var promotion chess.PieceType

	switch promo {
	case "rook":
		promotion = chess.PieceTypeRook
		break
	case "knight":
		promotion = chess.PieceTypeKnight
		break
	case "queen":
		promotion = chess.PieceTypeQueen
		break
	case "bishop":
		promotion = chess.PieceTypeBishop
		break
	default:
		s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf(
				"<@!%s> invalid promotion type",
				m.Author.ID,
			),
		)
		return
	}

	game, err := getGame(m, target)
	if err != nil {
		printMissingGame(s, m)
		return
	}

	game.MakeMove(chess.Move{
		From:      chess.StringToPostion(from),
		To:        chess.StringToPostion(to),
		Promotion: promotion,
	})
	dbIns.SaveGame(game)

	msg := fmt.Sprintf(
		"Match between <@!%s>: %s and <@!%s>: %s En Passant",
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

	resgin(s, m, target, game.GetOpponent(m.Author.ID).Side)
}

func resgin(s *discordgo.Session, m *discordgo.MessageCreate, target string, winner chess.SideType) {
	game, err := getGame(m, target)
	if err != nil {
		printMissingGame(s, m)
		return
	}

	game.Winner = winner

	err = dbIns.DeleteGame(game)
	if err != nil {
		s.ChannelMessageSend(
			m.ChannelID,
			fmt.Sprintf("<@!%s>: error deleting game! %v", m.Author.ID, err),
		)
	}
	go dbIns.ArchiveGame(game)

	var winID string
	if game.Winner == chess.SideWhite {
		winID = game.White.ID
	} else {
		winID = game.Black.ID
	}

	msg := fmt.Sprintf(
		"Match between <@!%s>: %s and <@!%s>: %s Final State\n"+
			"🎉Winner🎉 <@!%s>",
		game.White.ID, game.White.Side.String(), game.Black.ID, game.Black.Side.String(),
		winID,
	)
	s.ChannelMessageSendComplex(
		m.ChannelID,
		&discordgo.MessageSend{
			Content: msg,
			Files: []*discordgo.File{{
				Name: fmt.Sprintf("%s.gif", game.ID()), ContentType: "gif",
				Reader: game.CreateGif(),
			}},
		},
	)
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
}

func main() {
	fmt.Printf("Connecting to DB\n")
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

	discord.UpdateStatus(-1, "\"-cb help\"")

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	discord.Close()

}
