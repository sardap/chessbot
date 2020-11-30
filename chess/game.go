package chess

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/sardap/chessbot/env"
)

const (
	rowWidth = 8
	rowHight = 8
)

var (
	images     map[PieceType]map[SideType]image.Image = make(map[PieceType]map[SideType]image.Image)
	board      image.Image
	emptyBoard [rowHight][rowWidth]Piece
	green      = color.RGBA{0, 255, 0, 255}
	purple     = color.RGBA{255, 0, 255, 255}
)

func init() {
	// Loads assets
	board = changeColor(
		loadImage("assets/chess_board.png"),
		map[color.Color]color.Color{
			green:  color.RGBA{253, 209, 138, 255},
			purple: color.RGBA{137, 57, 34, 255},
		},
	)
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

func loadPieceImg(name string) map[SideType]image.Image {
	result := make(map[SideType]image.Image)
	result[SideWhite] = changeColor(
		loadImage(fmt.Sprintf("assets/%s.png", name)),
		map[color.Color]color.Color{
			green:  color.RGBA{255, 255, 255, 255},
			purple: color.RGBA{0, 0, 0, 255},
		},
	)
	result[SideBlack] = changeColor(
		loadImage(fmt.Sprintf("assets/%s.png", name)),
		map[color.Color]color.Color{
			green:  color.RGBA{0, 0, 0, 255},
			purple: color.RGBA{255, 255, 255, 255},
		},
	)
	return result
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
	return images[p.Kind][p.Side]
}

//Player player
type Player struct {
	ID   string   `json:"id"`
	Side SideType `json:"side"`
}

//Postion Postion
type Postion struct {
	Col int `json:"c"`
	Row int `json:"r"`
}

func (p *Postion) String() string {
	return fmt.Sprintf(
		"%s%d",
		string(p.Row+int('A')), 8-p.Col,
	)
}

//Move Move
type Move struct {
	From Postion `json:"from"`
	To   Postion `json:"to"`
}

//Game a chess game
type Game struct {
	board   [rowHight][rowWidth]Piece
	Moves   []Move   `json:"moves"`
	White   Player   `json:"white"`
	Black   Player   `json:"black"`
	GuildID string   `json:"gid"`
	Turn    SideType `json:"turn"`
	Winner  SideType `json:"win"`
}

//ID id
func (g *Game) ID() string {
	return fmt.Sprintf("%s_%s_%s", g.GuildID, g.White.ID, g.Black.ID)
}

func (g *Game) processMove(move Move) {
	tmp := g.board[move.From.Col][move.From.Row]
	g.board[move.From.Col][move.From.Row] = Piece{PieceTypeEmpty, SideEmpty}
	g.board[move.To.Col][move.To.Row] = tmp
}

//ProcessMoves process moves
func (g *Game) ProcessMoves() {
	g.board = emptyBoard

	for _, move := range g.Moves {
		g.processMove(move)
	}
}

func (g *Game) createImgRaw() image.Image {
	b := board.Bounds()
	snapshot := image.NewRGBA(b)
	draw.Draw(snapshot, b, board, image.ZP, draw.Src)

	for i := range g.board {
		for j, piece := range g.board[i] {
			if piece.Kind == PieceTypeEmpty {
				continue
			}

			offset := image.Pt(84+j*126, 84+i*126)
			img := piece.getImage()
			draw.Draw(snapshot, img.Bounds().Add(offset), img, image.ZP, draw.Over)
		}
	}

	return snapshot
}

//CreateImage CreateImage
func (g *Game) CreateImage() io.Reader {
	snapshot := g.createImgRaw()
	result := &bytes.Buffer{}

	jpeg.Encode(result, snapshot, &jpeg.Options{Quality: env.ImgQuality})
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

//ValidMove move
func (g *Game) ValidMove(id string, mv Move) bool {
	piece := g.board[mv.From.Col][mv.From.Row]
	if piece.Side != g.GetPlayer(id).Side {
		return false
	}

	if g.GetPlayer(id).Side != g.Turn {
		return false
	}

	switch piece.Kind {
	case PieceTypePawn:
		return true
		// if mv.From.Row != mv.To.Row {
		// 	return false
		// }

		// if piece.Side == SideWhite {
		// 	if mv.From.Col == rowHight-2 && mv.To.Col == mv.From.Col-2 {
		// 		return true
		// 	}

		// 	if mv.To.Col == mv.From.Col-1 {
		// 		return true
		// 	}
		// } else {
		// 	if mv.From.Col == 1 && mv.To.Col == mv.From.Col+2 {
		// 		return true
		// 	}

		// 	if mv.To.Col == mv.From.Col+1 {
		// 		return true
		// 	}
		// }
	}

	return true
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
	y := int(letter - 'A')
	x := int(number - '0')

	return Postion{
		Col: 8 - x,
		Row: y,
	}
}

func changeRowSide(row [rowWidth]Piece, side SideType) [rowWidth]Piece {
	for i := range row {
		row[i].Side = side
	}

	return row
}

//CreateGame CreateGame
func CreateGame(white, black, guildID string) Game {
	result := Game{
		White:   Player{ID: white, Side: SideWhite},
		Black:   Player{ID: black, Side: SideBlack},
		GuildID: guildID,
		Turn:    SideWhite,
	}

	result.ProcessMoves()

	return result
}
