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
	"math/rand"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"
)

const (
	rowWidth = 8
	rowHight = 8
	aiID     = "MR_COMPUTER"
)

type validMove func(g *Game, mv Move) error

var (
	//ErrCheckMate error when the player is checked mated
	ErrCheckMate = errors.New("checked mate")
	images       = make(map[PieceType]image.Image)
	moves        = make(map[PieceType]validMove)
	boardImg     image.Image
	emptyBoard   [rowHight][rowWidth]Piece
	green        = color.RGBA{0, 255, 0, 255}
	purple       = color.RGBA{255, 0, 255, 255}
	postions     = make([]Postion, rowHight*rowWidth)
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

	for i := 0; i < len(postions); i++ {
		postions[i] = Postion{
			Row: i / rowHight,
			Col: i % rowWidth,
		}
	}
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

//NotationStr returns algerbaric notation symbol for piece
func (p PieceType) NotationStr() string {
	switch p {
	case PieceTypePawn:
		return ""
	case PieceTypeKnight:
		return "K"
	case PieceTypeBishop:
		return "B"
	case PieceTypeRook:
		return "R"
	case PieceTypeQueen:
		return "Q"
	case PieceTypeKing:
		return "K"
	default:
		return "ERROR"
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

func colStr(c int) string {
	return fmt.Sprintf("%s", string(c+int('A')))
}

func rankStr(r int) string {
	return fmt.Sprintf("%d", 8-r)
}

//Move Move
type Move struct {
	From      Postion   `json:"from"`
	To        Postion   `json:"to"`
	Promotion PieceType `json:"promo"`
}

//BoardColor the colors for the board
type BoardColor struct {
	ColorWhite color.RGBA `json:"board_color_white"`
	ColorBlack color.RGBA `json:"board_color_black"`
}

//Game a chess game
type Game struct {
	board   [rowHight][rowWidth]Piece
	Moves   []Move     `json:"moves"`
	White   Player     `json:"white"`
	Black   Player     `json:"black"`
	GuildID string     `json:"gid"`
	Turn    SideType   `json:"turn"`
	Winner  SideType   `json:"win"`
	Color   BoardColor `json:"color"`
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

//AlgebraicNotation returns moves in atomic notation
func (g *Game) AlgebraicNotation() string {
	var result strings.Builder

	g.board = emptyBoard

	currentTurn := SideWhite
	for i, mv := range g.Moves {
		take := ""
		if g.getAt(mv.To).Kind != PieceTypeEmpty {
			take = "x"
		}

		postion := strings.ToLower(mv.To.String())

		moving := g.getAt(mv.From)
		pieces := g.findPieces(g.Turn, moving.Kind)
		//Checks if another piece of the same type can also make the move
		disambiguating := ""
		for _, val := range pieces {
			if val == mv.From {
				continue
			}

			err := g.validMoveForPiece(Move{
				From: val, To: mv.To,
			})
			if err == nil {
				if val.Col == mv.From.Col && val.Row == mv.From.Row {
					disambiguating = fmt.Sprintf("%s", strings.ToLower(mv.From.String()))
				} else if val.Col == mv.From.Col {
					disambiguating = fmt.Sprintf("%s", rankStr(val.Row))
				} else if val.Row == mv.From.Row {
					disambiguating = fmt.Sprintf("%s", colStr(val.Col))
				}
			}
		}

		fmt.Fprintf(
			&result, "%s%s%s%s ",
			moving.Kind.NotationStr(), disambiguating, take, postion,
		)
		if i != 0 && i%3 == 0 {
			fmt.Fprintf(&result, "\n")
		}
		if currentTurn == SideWhite {
			currentTurn = SideBlack
		} else {
			currentTurn = SideWhite
		}
		g.processMove(mv)
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

func (g *Game) validMoveForPiece(mv Move) error {
	source := g.getAt(mv.From)
	target := g.getAt(mv.To)

	if source.Side == target.Side {
		return errors.New("cannot kill a comrade with a move")
	}

	return moves[source.Kind](g, mv)
}

func validPawnMove(g *Game, mv Move) error {
	pawn := g.getAt(mv.From)

	//Check moving backwards
	if (pawn.Side == SideWhite && mv.To.Row >= mv.From.Row) ||
		(pawn.Side == SideBlack && mv.To.Row <= mv.From.Row) {
		return errors.New("pawns cannot take one step back")
	}

	target := g.getAt(mv.To)

	//Pawn Moving forward
	var maxMovement float64
	if pawn.Side == SideWhite && mv.From.Row == 6 ||
		pawn.Side == SideBlack && mv.From.Row == 1 {
		//Starting Rank
		maxMovement = 2
	} else {
		maxMovement = 1
	}

	if moveRowD := math.Abs(float64(mv.To.Row - mv.From.Row)); moveRowD > maxMovement {
		return errors.New("pawns cannot move that far ahead")
	}

	moveColD := math.Abs(float64(mv.To.Col - mv.From.Col))

	//Target is a enemy
	if target.Side == pawn.Side.other() {
		if moveColD != 1 {
			return errors.New("pawns cannot move diagonally without taking a piece")
		}

		return nil
	}

	//Checks if anyone is in the way
	if err := g.checkBlockingStraight(mv); err != nil {
		return err
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

func (g *Game) findPieces(side SideType, piece PieceType) []Postion {
	sidePieces := g.getPiecesForSide(side)
	result := []Postion{}
	for _, val := range sidePieces {
		if g.getAt(val).Kind == piece {
			result = append(result, val)
		}
	}

	return result
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
	kings := g.findPieces(side, PieceTypeKing)
	if len(kings) < 1 {
		return errors.Errorf("cannot find king")
	}

	kingPos := kings[0]
	other := g.getPiecesForSide(g.getAt(kingPos).Side.other())

	for _, val := range other {
		piece := g.getAt(val)
		err := g.validMoveForPiece(Move{
			From: val,
			To:   kingPos,
		})
		if err == nil {
			return errors.Errorf(
				"%s king is in check after move from %s: %s%s",
				side.String(), piece.Kind.String(),
				colStr(val.Col), rankStr(val.Row),
			)
		}
	}

	return nil
}

func (g *Game) movesIntoCheck(side SideType, mv Move) error {
	//Make move check if king is in check
	g.MakeMove(mv)
	defer func(g *Game) {
		//Remove added move and undo move
		g.Moves = g.Moves[0 : len(g.Moves)-1]
		g.board = emptyBoard
		g.ProcessMoves()
	}(g)

	if err := g.sideInCheck(side); err != nil {
		return err
	}

	return nil
}

func (g *Game) sideInCheckMate(side SideType, mv Move) error {
	//Make move check if king is in check
	g.MakeMove(mv)
	defer func(g *Game) {
		//Remove added move and undo move
		g.Moves = g.Moves[0 : len(g.Moves)-1]
		g.board = emptyBoard
		g.ProcessMoves()
	}(g)

	pieces := g.getPiecesForSide(side)
	for _, val := range pieces {
		for _, to := range postions {
			mv := Move{
				From: val,
				To:   to,
			}

			if err := g.validMoveForPiece(mv); err != nil {
				continue
			}

			if err := g.movesIntoCheck(side, mv); err == nil {
				return nil
			}
		}
	}

	return errors.Wrapf(ErrCheckMate, "player: %s", side.String())
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

	if err := g.validMoveForPiece(mv); err != nil {
		return err
	}

	if err := g.movesIntoCheck(piece.Side, mv); err != nil {
		return err
	}

	if err := g.sideInCheckMate(piece.Side.other(), mv); err != nil {
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
}

//NextTurn next turn
func (g *Game) NextTurn() {
	g.Turn = g.Turn.other()
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
		GuildID: guildID,
		Turn:    SideWhite,
		Color: BoardColor{
			ColorWhite: color.RGBA{255, 255, 255, 255},
			ColorBlack: color.RGBA{0, 0, 0, 255},
		},
	}

	result.ProcessMoves()

	return result
}

//CreateComputerGame creates a game vs a Computer
func CreateComputerGame(playerID, guildID string, whiteColor, BlackColor color.RGBA) Game {
	var white, black string
	if rand.Float32() > 0.5 {
		white = playerID
		black = aiID
	} else {
		white = aiID
		black = playerID
	}

	return CreateGame(white, black, guildID, whiteColor, BlackColor)
}
