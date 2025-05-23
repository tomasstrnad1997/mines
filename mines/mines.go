package mines

import (
	"fmt"
	"math/rand"
	"strconv"
)

type GameModeId byte

const (
	ModeClassic GameModeId = 0
	ModeCoop               = 1
)


var GameModeNames = map[GameModeId]string{
	ModeClassic: "Classic",
	ModeCoop:    "Coop",
}

type Cell struct {
	Mine     bool
	Revealed bool
	Flagged  bool
	X        int
	Y        int
}

type GamemodeUpdateInfo interface {
	GetGameModeId() GameModeId
}

type Board struct {
	Width         int
	Height        int
	Mines         int
	Cells         [][]*Cell
	RevealedCells int
}

type MoveType byte

const (
	Reveal MoveType = 0x01
	Flag            = 0x02
)

type Move struct {
	X        int
	Y        int
	Type     MoveType
	PlayerId uint32
}

type GameParams struct {
	Width    int
	Height   int
	Mines    int
	GameMode GameModeId
}

type GameMode interface {
	Init(*Board)
	Name() string
	GameModeId() GameModeId
	OnMove(*Board, Move, *MoveResult) (GamemodeUpdateInfo, error) // Returns the changes to the gamemode
}

type Game struct {
	board  *Board
	Params GameParams
	Mode   GameMode
}

func (game *Game) MakeMove(move Move) (*MoveResult, GamemodeUpdateInfo, error) {
	result, err := game.board.makeMove(move)
	if err != nil {
		return nil, nil, err
	}
	deltaState, err := game.Mode.OnMove(game.board, move, result)
	if err != nil {
		return nil, nil, err
	}

	return result, deltaState, err

}

func GetGameModeById(id GameModeId) (GameMode, error) {
	switch id {
	case ModeClassic:
		return &Classic{}, nil
	case ModeCoop:
		return &Coop{}, nil
	default:
		return nil, fmt.Errorf("Unknown gamemode id: %d", id)
	}
}

func CreateGame(params GameParams) (*Game, error) {
	board, err := CreateBoardFromParams(params)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	gamemode, err := GetGameModeById(params.GameMode)
	if err != nil {
		return nil, err
	}
	gamemode.Init(board)

	return &Game{board: board, Params: params, Mode: gamemode}, nil
}

func (move Move) String() string {
	msg := fmt.Sprintf("(%d, %d) ", move.X, move.Y)
	switch move.Type {
	case Reveal:
		return msg + "Reveal"
	case Flag:
		return msg + "Flag"
	default:
		return msg + "UNKNOWN"
	}
}

type InvalidBoardParamsError struct {
	height int
	width  int
	mines  int
}

type InvalidMoveError struct {
	board *Board
	x     int
	y     int
}

type MoveResultType int

const (
	NoChange MoveResultType = iota
	MineBlown
	CellRevealed
	Flagged
	GameWon
)

type MoveResult struct {
	Result       MoveResultType
	UpdatedCells []*Cell
}

const (
	ShowCount byte = 0x00
	ShowMine       = 0x10
	ShowFlag       = 0x20
	Unflag         = 0x30
)

type UpdatedCell struct {
	X     int
	Y     int
	Value byte
}

func (e InvalidMoveError) Error() string {
	return fmt.Sprintf("Move out of range - (%d, %d) - Board (%d, %d)", e.x, e.y, e.board.Width, e.board.Height)
}

func (e InvalidBoardParamsError) Error() string {
	switch {
	case e.width <= 0:
		return fmt.Sprintf("Cannot create a board with width: %d", e.width)
	case e.height <= 0:
		return fmt.Sprintf("Cannot create a board with height: %d", e.height)
	case e.mines < 0:
		return fmt.Sprintf("Cannot create a board with negative amount of mines: %d", e.mines)
	case e.mines > e.width*e.height:
		return fmt.Sprintf("Not enough space for %d mines. (%d > %d * %d)", e.mines, e.mines, e.width, e.height)
	default:
		return "Cannot construct board: unknown error"
	}
}

func CreateBoardFromParams(params GameParams) (*Board, error) {
	return CreateBoard(params.Width, params.Height, params.Mines)
}

func CreateBoard(width, height, mines int) (*Board, error) {
	if (width <= 0) || (height <= 0) || (mines < 0) || (mines > width*height) {
		return nil, &InvalidBoardParamsError{height, width, mines}

	}
	cells := make([][]*Cell, width)
	for i := range cells {
		cells[i] = make([]*Cell, height)
		for j := range height {
			cells[i][j] = &Cell{false, false, false, i, j}
		}
	}
	mines_position := make([]int, width*height)

	for i := range mines_position {
		mines_position[i] = i
	}

	rand.Shuffle(len(mines_position), func(i, j int) {
		mines_position[i], mines_position[j] = mines_position[j], mines_position[i]
	})
	for _, position := range mines_position[:mines] {
		cells[position%width][position/width].Mine = true
	}

	return &Board{width, height, mines, cells, 0}, nil

}

func cascade(board *Board, cell *Cell, updatedCells []*Cell) []*Cell {
	cell.Revealed = true
	updatedCells = append(updatedCells, cell)

	if GetNumberOfMines(board, cell) != 0 {
		return updatedCells
	}
	for _, ncell := range getNeighbouringCells(board, cell) {
		if !ncell.Revealed && !ncell.Flagged {
			updatedCells = cascade(board, ncell, updatedCells)
		}
	}
	return updatedCells
}

func ValidCellIndex(board *Board, x, y int) bool {
	return (x >= 0) && (x < board.Width) && (y >= 0) && (y < board.Height)
}

func (board *Board) Reveal(x, y int) (*MoveResult, error) {
	if !ValidCellIndex(board, x, y) {
		return nil, &InvalidMoveError{board, x, y}
	}
	var cell = board.Cells[x][y]
	if cell.Revealed || cell.Flagged {
		return &MoveResult{NoChange, nil}, nil
	}
	if cell.Mine {
		cell.Revealed = true
		return &MoveResult{MineBlown, []*Cell{cell}}, nil
	}
	var updatedCells = []*Cell{}
	updatedCells = cascade(board, cell, updatedCells)
	board.RevealedCells += len(updatedCells)
	var result MoveResultType
	if board.RevealedCells+board.Mines == board.Width*board.Height {
		result = GameWon
	} else {
		result = CellRevealed
	}

	return &MoveResult{result, updatedCells}, nil
}

func getNeighbouringCells(board *Board, cell *Cell) []*Cell {
	var cells []*Cell
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			x := cell.X + dx
			y := cell.Y + dy
			if ValidCellIndex(board, x, y) {
				cells = append(cells, board.Cells[x][y])
			}
		}
	}
	return cells

}

func GetNumberOfMines(board *Board, cell *Cell) int {
	mines := 0
	for _, cell := range getNeighbouringCells(board, cell) {
		if cell.Mine {
			mines++
		}
	}
	return mines

}

func (board *Board) Print() {
	print("X")
	for i := range board.Width {
		print(i % 10)
	}
	println()
	for y := range board.Height {
		print(y % 10)
		for x := range board.Width {
			if board.Cells[x][y].Revealed {
				print(strconv.Itoa(GetNumberOfMines(board, board.Cells[x][y])))
			} else if board.Cells[x][y].Flagged {
				print("F")

			} else {
				print("#")
			}
		}
		println()
	}

}
func (board *Board) PrintRevaled() {
	for y := range board.Height {
		for x := range board.Width {
			if board.Cells[x][y].Mine {
				print("O")
			} else {
				print("#")
			}
		}
		println()
	}

}
func (board *Board) RemainingCells() int {
	remaining := 0
	for _, column := range board.Cells {
		for _, cell := range column {
			if !cell.Revealed {
				remaining++
			}
		}
	}
	return remaining
}

func (board *Board) Flag(x, y int) (*MoveResult, error) {

	if !ValidCellIndex(board, x, y) {
		return nil, &InvalidMoveError{board, x, y}
	}
	if board.Cells[x][y].Revealed {
		return &MoveResult{NoChange, nil}, nil
	}
	board.Cells[x][y].Flagged = !board.Cells[x][y].Flagged
	return &MoveResult{Flagged, []*Cell{board.Cells[x][y]}}, nil
}

func (board *Board) makeMove(move Move) (*MoveResult, error) {
	switch move.Type {
	case Reveal:
		return board.Reveal(move.X, move.Y)
	case Flag:
		return board.Flag(move.X, move.Y)
	default:
		return nil, fmt.Errorf("Invalid move type %x", move.Type)

	}
}

func (board *Board) ProcessTextCommand(text string) (*MoveResult, error) {
	var x, y int
	var flag rune
	flag = 'X'
	n, _ := fmt.Sscanf(text, "%d %d %c\n", &x, &y, &flag)
	if n < 2 {
		println(n)
		return nil, fmt.Errorf("Incorrect input")
	}
	if flag == 'f' || flag == 'F' {
		return board.Flag(x, y)
	} else {
		result, err := board.Reveal(x, y)
		println(result.UpdatedCells)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
}

func (game *Game) CreateCellUpdates(cells []*Cell) ([]UpdatedCell, error) {
	return game.board.CreateCellUpdates(cells)
}

func (board *Board) CreateCellUpdates(cells []*Cell) ([]UpdatedCell, error) {
	updates := make([]UpdatedCell, len(cells))
	var value byte
	for i, cell := range cells {
		if cell.Revealed {
			if cell.Mine {
				value = ShowMine
			} else {
				value = (byte(GetNumberOfMines(board, cell)))
			}
		} else if cell.Flagged {
			value = ShowFlag
		} else {
			// Is not flagger nor revealed so it must be unflag
			value = Unflag
		}
		updates[i] = UpdatedCell{X: cell.X, Y: cell.Y, Value: value}
	}
	return updates, nil
}

func (game *Game) GetChangedCellUpdates() ([]UpdatedCell, error) {
	return game.board.GetChangedCellUpdates()
}

func (board *Board) GetChangedCellUpdates() ([]UpdatedCell, error) {
	updatedCells := []*Cell{}
	for y := range board.Height {
		for x := range board.Width {
			cell := board.Cells[x][y]
			if cell.Revealed || cell.Flagged {
				updatedCells = append(updatedCells, cell)
			}
		}
	}
	return board.CreateCellUpdates(updatedCells)
}
