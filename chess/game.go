package chess

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/png"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

const (
	rowWidth = 8
	rowHight = 8
)

var (
	images     map[PieceType]image.Image = make(map[PieceType]image.Image)
	boardImg   image.Image
	emptyBoard [rowHight][rowWidth]Piece
	green      = color.RGBA{0, 255, 0, 255}
	purple     = color.RGBA{255, 0, 255, 255}
)

func init() {
	// Loads assets
	boardImg = loadImage("assets/chess_board.png")

	images[PieceTypePawn] = loadPieceImg("pawn")
	images[PieceTypeKnight] = loadPieceImg("knight")
	images[PieceTypeBishop] = loadPieceImg("bishop")
	images[PieceTypeRook] = loadPieceImg("rook")
	images[PieceTypeQueen] = loadPieceImg("queen")
	images[PieceTypeKing] = loadPieceImg("king")

	var board [rowHight][rowWidth]Piece

	coolPieceRow := [rowWidth]Piece{
		{PieceTypeRook, SideEmpty}, {PieceTypeKnight, SideEmpty}, {PieceTypeBishop, SideEmpty},
		{PieceTypeQueen, SideEmpty}, {PieceTypeKing, SideEmpty}, {PieceTypeBishop, SideEmpty},
		{PieceTypeKnight, SideEmpty}, {PieceTypeRook, SideEmpty},
	}
	pawnRow := [rowWidth]Piece{
		{PieceTypePawn, SideEmpty}, {PieceTypePawn, SideEmpty}, {PieceTypePawn, SideEmpty},
		{PieceTypePawn, SideEmpty}, {PieceTypePawn, SideEmpty}, {PieceTypePawn, SideEmpty},
		{PieceTypePawn, SideEmpty}, {PieceTypePawn, SideEmpty},
	}

	// Setup Black
	board[0] = changeRowSide(coolPieceRow, SideBlack)
	board[1] = changeRowSide(pawnRow, SideBlack)

	// Setup White
	board[rowHight-2] = changeRowSide(pawnRow, SideWhite)
	board[rowHight-1] = changeRowSide(coolPieceRow, SideWhite)

	emptyBoard = board
}

func loadImage(path string) image.Image {
	file, err := os.Open(path)
	if err != nil {
		panic(errors.Wrapf(err, " file: %s", path))
	}
	defer file.Close()
	result, err := png.Decode(file)
	if err != nil {
		panic(err)
	}

	return result
}

func changeColor(src image.Image, target map[color.Color]color.Color) image.Image {
	b := src.Bounds()
	m := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for x := 0; x < b.Dx(); x++ {
		for y := 0; y < b.Dy(); y++ {
			r, g, b, a := src.At(x, y).RGBA()
			var srcColor color.Color
			srcColor = color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a)}
			if val, ok := target[srcColor]; ok {
				srcColor = val
			}
			m.Set(x, y, srcColor)
		}
	}

	return m
}

func loadPieceImg(name string) image.Image {
	return loadImage(fmt.Sprintf("assets/%s.png", name))
}

//PieceType PieceType
type PieceType int

const (
	//PieceTypeEmpty PieceTypeEmpty
	PieceTypeEmpty PieceType = iota
	//PieceTypePawn pawn
	PieceTypePawn
	//PieceTypeKnight Knight
	PieceTypeKnight
	//PieceTypeBishop Bishop
	PieceTypeBishop
	//PieceTypeRook Rook
	PieceTypeRook
	//PieceTypeQueen Queen
	PieceTypeQueen
	//PieceTypeKing King
	PieceTypeKing
)

//SideType SideType
type SideType int

func (s SideType) String() string {
	switch s {
	case SideWhite:
		return "White"
	case SideBlack:
		return "Black"
	}

	return "ERROR"
}

const (
	//SideEmpty SideEmpty
	SideEmpty SideType = iota
	//SideWhite SideWhite
	SideWhite
	//SideBlack SideBlack
	SideBlack
)

//Piece Piece
type Piece struct {
	Kind PieceType `json:"kind"`
	Side SideType  `json:"side"`
}

func (p *Piece) getImage() image.Image {
	return images[p.Kind]
}

//Player player
type Player struct {
	ID    string     `json:"id"`
	Side  SideType   `json:"side"`
	Color color.RGBA `json:"color"`
}

//Postion Postion
type Postion struct {
	Y int `json:"c"`
	X int `json:"r"`
}

func (p *Postion) String() string {
	return fmt.Sprintf(
		"%s%d",
		string(p.X+int('A')), 8-p.Y,
	)
}

//Move Move
type Move struct {
	From      Postion   `json:"from"`
	To        Postion   `json:"to"`
	Promotion PieceType `json:"promo"`
}

//Game a chess game
type Game struct {
	board           [rowHight][rowWidth]Piece
	Moves           []Move     `json:"moves"`
	White           Player     `json:"white"`
	Black           Player     `json:"black"`
	GuildID         string     `json:"gid"`
	Turn            SideType   `json:"turn"`
	Winner          SideType   `json:"win"`
	BoardColorWhite color.RGBA `json:"board_color_white"`
	BoardColorBlack color.RGBA `json:"board_color_black"`
}

//GameID Create game id
func GameID(guild, id1, id2 string) string {
	ary := []string{id1, id2}
	sort.Strings(ary)
	return fmt.Sprintf("%s_%s_%s", guild, ary[0], ary[1])
}

//ID id
func (g *Game) ID() string {
	return GameID(g.GuildID, g.White.ID, g.Black.ID)
}

func (g *Game) processMove(move Move) {
	tmp := g.board[move.From.Y][move.From.X]
	g.board[move.From.Y][move.From.X] = Piece{PieceTypeEmpty, SideEmpty}
	g.board[move.To.Y][move.To.X] = tmp
	//Apply promotion
	if move.Promotion != PieceTypeEmpty {
		g.board[move.To.Y][move.To.X] = Piece{
			move.Promotion, g.board[move.To.Y][move.To.X].Side,
		}
	}
}

//ProcessMoves process moves
func (g *Game) ProcessMoves() {
	g.board = emptyBoard

	for _, move := range g.Moves {
		g.processMove(move)
	}
}

func (g *Game) createImgRaw() image.Image {
	boardImgColored := changeColor(
		boardImg,
		map[color.Color]color.Color{
			green:  color.RGBA{253, 209, 138, 255},
			purple: color.RGBA{137, 57, 34, 255},
		},
	)

	b := boardImgColored.Bounds()
	snapshot := image.NewRGBA(b)
	draw.Draw(snapshot, b, boardImgColored, image.ZP, draw.Src)

	whiteColor := map[color.Color]color.Color{
		green:  g.White.Color,
		purple: g.Black.Color,
	}

	blackColor := map[color.Color]color.Color{
		green:  g.Black.Color,
		purple: g.White.Color,
	}

	for i := range g.board {
		for j, piece := range g.board[i] {
			if piece.Kind == PieceTypeEmpty {
				continue
			}

			var pal map[color.Color]color.Color
			if piece.Side == SideWhite {
				pal = whiteColor
			} else {
				pal = blackColor
			}

			img := changeColor(piece.getImage(), pal)
			offset := image.Pt(84+j*126, 84+i*126)
			draw.Draw(snapshot, img.Bounds().Add(offset), img, image.ZP, draw.Over)
		}
	}

	return snapshot
}

//CreateImage CreateImage
func (g *Game) CreateImage() io.Reader {
	snapshot := g.createImgRaw()
	result := &bytes.Buffer{}

	png.Encode(result, snapshot)
	return result
}

//CreateGif Creates a jif of all the moves
func (g *Game) CreateGif() io.Reader {
	//Reset board
	g.board = emptyBoard

	var tmpMoves []Move
	//Append empty move so start board state is shown
	tmpMoves = append(tmpMoves, Move{})
	tmpMoves = append(tmpMoves, g.Moves...)

	snapshots := make([]*image.Paletted, len(tmpMoves))
	var delays []int

	type snapshot struct {
		img *image.Paletted
		idx int
	}

	piCh := make(chan snapshot, len(tmpMoves)-1)

	for i, move := range tmpMoves {
		g.processMove(move)

		simage := g.createImgRaw()

		go func(simage image.Image, idx int, ch chan snapshot) {
			bounds := simage.Bounds()
			palettedImage := image.NewPaletted(bounds, palette.Plan9)
			draw.Draw(palettedImage, palettedImage.Rect, simage, bounds.Min, draw.Over)
			ch <- snapshot{palettedImage, idx}
		}(simage, i, piCh)
	}

	complete := false
	for !complete {
		top := <-piCh
		snapshots[top.idx] = top.img
		delays = append(delays, 100)
		//Bad
		complete = true
		for _, val := range snapshots {
			if val == nil {
				complete = false
				break
			}
		}
	}

	result := &bytes.Buffer{}

	anim := gif.GIF{Delay: delays, Image: snapshots}

	gif.EncodeAll(result, &anim)

	return result
}

//MovesAtomicNotation returns moves in atomic notation
func (g *Game) MovesAtomicNotation() string {
	var result strings.Builder

	currentTurn := SideWhite
	for i, mv := range g.Moves {
		fmt.Fprintf(&result, "%s %s: %s ", mv.From.String(), mv.To.String(), currentTurn.String())
		if i != 0 && i%3 == 0 {
			fmt.Fprintf(&result, "\n")
		}
		if currentTurn == SideWhite {
			currentTurn = SideBlack
		} else {
			currentTurn = SideWhite
		}
	}

	return result.String()
}

//GetPlayer GetPlayer
func (g *Game) GetPlayer(id string) Player {
	if g.White.ID == id {
		return g.White
	}
	return g.Black
}

//GetOpponent GetOpponent
func (g *Game) GetOpponent(id string) Player {
	if g.White.ID != id {
		return g.White
	}
	return g.Black
}

func (g *Game) getAt(pos Postion) Piece {
	return g.board[pos.Y][pos.X]
}

func (g *Game) diagonalMove(mv Move) bool {
	return mv.To.X != mv.From.X && mv.To.Y != mv.From.Y
}

func (g *Game) validPawnMove(mv Move) error {

	if mv.From.X != mv.To.X {
		return errors.New("cannot move the other players pieces")
	}

	return nil
}

//ValidMove move
func (g *Game) ValidMove(id string, mv Move) error {
	if g.GetPlayer(id).Side != g.Turn {
		return errors.New("cannot move on enemies turn")
	}

	piece := g.board[mv.From.Y][mv.From.X]
	if piece.Side != g.GetPlayer(id).Side {
		return errors.New("cannot move the other players pieces")
	}

	switch piece.Kind {
	case PieceTypePawn:
		return nil
	}

	return nil
}

//FindEmptySqaure This is used for one hell of  a hack
func (g *Game) FindEmptySqaure() Postion {
	for i := range g.board {
		for j := range g.board[i] {
			if g.board[i][j].Kind == PieceTypeEmpty {
				return Postion{i, j}
			}
		}
	}

	return Postion{0, 0}
}

//MakeMove move
func (g *Game) MakeMove(mv Move) {
	g.Moves = append(g.Moves, mv)
	g.processMove(mv)

	if g.Turn == SideWhite {
		g.Turn = SideBlack
	} else {
		g.Turn = SideWhite
	}
}

//StringToPostion StringToPostion
func StringToPostion(from string) Postion {
	letter := from[0]
	number := from[1]
	y := int(letter - 'a')
	x := int(number - '0')

	return Postion{
		Y: 8 - x,
		X: y,
	}
}

func changeRowSide(row [rowWidth]Piece, side SideType) [rowWidth]Piece {
	for i := range row {
		row[i].Side = side
	}

	return row
}

//CreateGame CreateGame
func CreateGame(white, black, guildID string, whiteColor, BlackColor color.RGBA) Game {
	result := Game{
		White: Player{
			ID:    white,
			Side:  SideWhite,
			Color: whiteColor,
		},
		Black: Player{
			ID:    black,
			Side:  SideBlack,
			Color: BlackColor,
		},
		GuildID:         guildID,
		Turn:            SideWhite,
		BoardColorBlack: color.RGBA{0, 0, 0, 255},
		BoardColorWhite: color.RGBA{255, 255, 255, 255},
	}

	result.ProcessMoves()

	return result
}
