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
	"math"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

const (
	rowWidth = 8
	rowHight = 8
)

type validMove func(g *Game, mv Move) error

var (
	images     = make(map[PieceType]image.Image)
	moves      = make(map[PieceType]validMove)
	boardImg   image.Image
	emptyBoard [rowHight][rowWidth]Piece
	green      = color.RGBA{0, 255, 0, 255}
	purple     = color.RGBA{255, 0, 255, 255}
)

func init() {
	moves = map[PieceType]validMove{
		PieceTypePawn:   validPawnMove,
		PieceTypeKnight: validHorsieMove,
		PieceTypeBishop: validBishopMove,
		PieceTypeRook:   validRookMove,
		PieceTypeQueen:  validQueenMove,
		PieceTypeKing:   validKingMove,
	}

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

func (p PieceType) String() string {
	switch p {
	case PieceTypePawn:
		return "pawn"
	case PieceTypeKnight:
		return "knight"
	case PieceTypeBishop:
		return "bishop"
	case PieceTypeRook:
		return "rook"
	case PieceTypeQueen:
		return "queen"
	case PieceTypeKing:
		return "king"
	default:
		return "comrade"
	}
}

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

func (s SideType) other() SideType {
	switch s {
	case SideWhite:
		return SideBlack
	case SideBlack:
		return SideWhite
	}

	return SideEmpty
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
	Row int `json:"c"`
	Col int `json:"r"`
}

func (p *Postion) String() string {
	return fmt.Sprintf(
		"%s%d",
		string(p.Col+int('A')), 8-p.Row,
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
	tmp := g.board[move.From.Row][move.From.Col]
	g.board[move.From.Row][move.From.Col] = Piece{PieceTypeEmpty, SideEmpty}
	g.board[move.To.Row][move.To.Col] = tmp
	//Apply promotion
	if move.Promotion != PieceTypeEmpty {
		g.board[move.To.Row][move.To.Col] = Piece{
			move.Promotion, g.board[move.To.Row][move.To.Col].Side,
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
	return g.board[pos.Row][pos.Col]
}

func (g *Game) diagonalMove(mv Move) bool {
	return mv.To.Col != mv.From.Col && mv.To.Row != mv.From.Row
}

func (g *Game) checkRoute(mv Move) error {
	step := func() {
		if mv.From.Col > mv.To.Col {
			mv.From.Col--
		}

		if mv.From.Col < mv.To.Col {
			mv.From.Col++
		}

		if mv.From.Row > mv.To.Row {
			mv.From.Row--
		}

		if mv.From.Row < mv.To.Row {
			mv.From.Row++
		}
	}

	step()
	loops := 0
	for loops < 256 && (mv.From.Col != mv.To.Col || mv.From.Row != mv.To.Row) {
		if g.getAt(mv.From).Kind != PieceTypeEmpty {
			return errors.Errorf("piece in the way of %s movement", g.getAt(mv.From).Kind.String())
		}

		loops++
		step()
	}

	return nil
}

func (g *Game) checkBlockingStraight(mv Move) error {
	colD := math.Abs(float64(mv.To.Col - mv.From.Col))
	rowD := math.Abs(float64(mv.To.Row - mv.From.Row))

	if rowD != 0 && colD != 0 {
		return errors.Errorf("Cannot move %s dialoginly", g.getAt(mv.From).Kind.String())
	}

	//Don't need to check for one space moves
	if colD <= 1 && rowD <= 1 {
		return nil
	}

	return g.checkRoute(mv)
}

func (g *Game) checkBlockingDiagonal(mv Move) error {
	colD := math.Abs(float64(mv.To.Col - mv.From.Col))
	rowD := math.Abs(float64(mv.To.Row - mv.From.Row))

	if rowD == 0 || colD == 0 {
		return errors.Errorf("Cannot move %s straight", g.getAt(mv.From).Kind.String())
	}

	if rowD != colD {
		return errors.Errorf("illegal movement cell for %s", g.getAt(mv.From).Kind.String())
	}

	//Don't need to check for one space moves
	if colD <= 1 && rowD <= 1 {
		return nil
	}

	return g.checkRoute(mv)
}

func validPawnMove(g *Game, mv Move) error {
	pawn := g.getAt(mv.From)
	target := g.getAt(mv.To)

	if err := g.checkBlockingStraight(mv); err != nil {
		return err
	}

	moveColD := math.Abs(float64(mv.To.Col - mv.From.Col))

	// Taking another piece
	if target.Kind == PieceTypeEmpty && mv.From.Col != mv.To.Col && moveColD > 1 {
		return errors.New("pawns cannot move diagonally without taking a piece")
	}

	moveD := math.Abs(float64(mv.To.Row - mv.From.Row))
	if moveD <= 0 {
		return errors.New("pawns cannot move backwards or to the same cell")
	}

	var maxMovement float64
	if pawn.Side == SideWhite && mv.From.Row == 6 ||
		pawn.Side == SideBlack && mv.From.Row == 1 {
		maxMovement = 2
	} else {
		maxMovement = 1
	}

	//Moving forward
	if moveD > maxMovement {
		return errors.New("pawns cannot move that far ahead")
	}

	return nil
}

func validBishopMove(g *Game, mv Move) error {
	return g.checkBlockingDiagonal(mv)
}

func validHorsieMove(g *Game, mv Move) error {
	colMoveD := math.Abs(float64(mv.To.Col - mv.From.Col))
	rowMoveD := math.Abs(float64(mv.To.Row - mv.From.Row))

	if colMoveD == 1 && rowMoveD == 2 || colMoveD == 2 && rowMoveD == 1 {
		return nil
	}

	return errors.New("invalid knight move")
}

func validRookMove(g *Game, mv Move) error {
	return g.checkBlockingStraight(mv)
}

func validQueenMove(g *Game, mv Move) error {
	straightErr := g.checkBlockingStraight(mv)
	diagonalErr := g.checkBlockingDiagonal(mv)

	if straightErr != nil && diagonalErr != nil {
		return errors.New("queen cannot move to that postion")
	}

	return nil
}

func validKingMove(g *Game, mv Move) error {
	colMoveD := math.Abs(float64(mv.To.Col - mv.From.Col))
	rowMoveD := math.Abs(float64(mv.To.Row - mv.From.Row))

	if colMoveD > 1 || rowMoveD > 1 {
		return errors.New("king cannot move more then one cell")
	}

	return nil
}

func (g *Game) findPiece(side SideType, piece PieceType) Postion {
	sidePieces := g.getPiecesForSide(side)
	for _, val := range sidePieces {
		if g.getAt(val).Kind == piece {
			return val
		}
	}

	return Postion{}
}

func (g *Game) getPiecesForSide(side SideType) []Postion {
	results := make([]Postion, 0)
	for r := 0; r < rowWidth; r++ {
		for c := 0; c < rowHight; c++ {
			postion := Postion{Row: r, Col: c}
			if g.getAt(postion).Side == side {
				results = append(results, postion)
			}
		}
	}
	return results
}

func (g *Game) sideInCheck(side SideType) error {
	kingPos := g.findPiece(side, PieceTypeKing)
	other := g.getPiecesForSide(g.getAt(kingPos).Side.other())

	for _, val := range other {
		err := moves[g.getAt(val).Kind](g, Move{
			From: val,
			To:   kingPos,
		})
		if err == nil {
			return errors.Errorf("%s king is in check after move", side.String())
		}
	}

	return nil
}

//ValidMove returns an error if the move is not valid
func (g *Game) ValidMove(id string, mv Move) error {
	if g.GetPlayer(id).Side != g.Turn {
		return errors.New("cannot move on enemies turn")
	}

	piece := g.board[mv.From.Row][mv.From.Col]
	if piece.Side != g.GetPlayer(id).Side {
		return errors.New("cannot move the other players pieces")
	}

	target := g.getAt(mv.To)

	if target.Side == piece.Side {
		return errors.New("cannot kill a comrade with a move")
	}

	if err := moves[piece.Kind](g, mv); err != nil {
		return err
	}

	//Make move check if king is in check
	g.MakeMove(mv)
	defer func(g *Game) {
		//Remove added move and undo move
		g.Moves = g.Moves[0 : len(g.Moves)-1]
		g.board = emptyBoard
		g.ProcessMoves()
	}(g)

	if err := g.sideInCheck(g.Turn); err != nil {
		return err
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
		Row: 8 - x,
		Col: y,
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
