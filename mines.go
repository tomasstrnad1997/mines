package main

import (
	"fmt"
	"math/rand"
	"strconv"
)

type Cell struct
{
    mine bool
    revealed bool
    flagged bool
    x int 
    y int

}

type Board struct {
    height int
    width int
    cells [][]*Cell

}

type InvalidBoardParamsError struct {
    height int
    width int
    mines int
}

type InvalidMoveError struct {
    board *Board
    x int
    y int
}

type MoveResultType int

const (
    NoChange MoveResultType = iota
    MineBlown
    CellRevealed
)

type MoveResult struct {
    Result MoveResultType
    UpdatedCells []*Cell
}

func (e InvalidMoveError) Error() string {
    return fmt.Sprintf("Move out of range - (%d, %d) - Board (%d, %d)", e.x, e.y, e.board.width, e.board.height)
}

func (e InvalidBoardParamsError) Error() string {
    switch {
    case e.width <= 0:
        return fmt.Sprintf("Cannot create a board with width: %d", e.width)
    case e.height <= 0:
        return fmt.Sprintf("Cannot create a board with height: %d", e.height)
    case e.mines < 0:
        return fmt.Sprintf("Cannot create a board with negative amount of mines: %d", e.mines)
    case e.mines > e.width * e.height:
        return fmt.Sprintf("Not enough space for %d mines. (%d > %d * %d)", e.mines, e.mines, e.width, e.height)
    default:
        return "Cannot construct board: unknown error"
    }
}

func CreateBoard(width, height, mines int) (*Board, error) {
    if (width <= 0) || (height <= 0) || (mines < 0) || (mines > width * height) {
        return nil, &InvalidBoardParamsError{height, width, mines}

    }
    cells := make([][]*Cell, width)
    for i := range cells {
        cells[i] = make([]*Cell, height)
        for j := 0; j < height; j++{
            cells[i][j] = &Cell{false, false, false, i, j}
        }
    }
    mines_position := make([]int, width * height)
    
    for i := range mines_position{
        mines_position[i] = i;
    }
    
    rand.Shuffle(len(mines_position), func (i, j int) {
        mines_position[i], mines_position[j] = mines_position[j], mines_position[i]
    })
    for _, position := range mines_position[:mines]{
        cells[position / width][position % height].mine = true;
    }


    return &Board{width, height, cells}, nil

}


func Cascade(board *Board, cell *Cell, updatedCells []*Cell) ([]*Cell){
    cell.revealed = true
    updatedCells = append(updatedCells, cell)

    if GetNumberOfMines(board, cell) != 0 {
        return updatedCells
    }
    for _, ncell := range GetNeighbouringCells(board, cell){
        if !ncell.revealed {
            Cascade(board, ncell, updatedCells)
        }
    }
    return updatedCells
}
func ValidCellIndex(board *Board, x, y int) bool {
    return !(x < 0 || x >= board.width || y >= board.height || y < 0)
}

func (board *Board) MakeMove(x, y int) (*MoveResult, error) {
    if !ValidCellIndex(board, x, y){
        return nil, &InvalidMoveError{board, x, y};
    }
    var cell = board.cells[x][y]
    if cell.revealed || cell.flagged{
        return &MoveResult{NoChange, nil}, nil
    }
    if cell.mine {
        return &MoveResult{MineBlown, []*Cell{cell}}, nil
    }
    var updatedCells = []*Cell{}
    Cascade(board, cell, updatedCells)
    return &MoveResult{CellRevealed, updatedCells}, nil
}

func GetNeighbouringCells(board *Board, cell *Cell) []*Cell {
    var cells []*Cell
    for dx := -1; dx <= 1; dx++{
        for dy := -1; dy <= 1; dy++{
            x := cell.x + dx
            y := cell.y + dy
            if ValidCellIndex(board, x, y){
                cells = append(cells, board.cells[x][y])
            }
        }
    }
    return cells

}

func GetNumberOfMines(board *Board, cell *Cell) int {
    mines := 0 
    for _, cell := range GetNeighbouringCells(board, cell){
        if cell.mine {
            mines ++
        }
    }
    return mines

}

func (board *Board) Print() {
    print("X")
    for i:=0; i < board.width; i++{
        print(i % 10)
    }
    println()
    for y := 0; y < board.height; y++{
        print(y % 10)
        for x := 0; x < board.width; x++{
            if board.cells[x][y].revealed{
                print(strconv.Itoa(GetNumberOfMines(board, board.cells[x][y])))
            }else if board.cells[x][y].flagged{
                print("F")

            }else{
                print("#")
            }
        }
        println()
    }

}
func (board *Board) PrintRevaled() {
    for y := 0; y < board.height; y++{
        for x := 0; x < board.width; x++{
            if board.cells[x][y].mine{
                print("O")
            }else{
                print("#")
            }
        }
        println()
    }

}
func (board *Board) RemainingCells() int {
    remaining := 0
    for _, column := range board.cells{
        for _, cell := range column{
            if !cell.revealed {
                remaining++
            }
        }
    }
    return remaining
}

func (board *Board) Flag(x, y int) error {
    
    if !ValidCellIndex(board, x, y){
        return &InvalidMoveError{board, x, y};
    }
    board.cells[x][y].flagged = !board.cells[x][y].flagged
    return nil
}

func main() {
    mines := 5
    board, err := CreateBoard(5, 5, mines);
    if err != nil {
        fmt.Println(err)
        return
    }
    var x, y int
    var flag rune
    for {
        println("****************")
        remaining := board.RemainingCells()
        fmt.Printf("%d-%d\n", remaining, mines)
        board.Print()
        if remaining == mines{
            println("CLEARED")
            return
        }
        flag = 'X'
        n, _ := fmt.Scanf("%d %d %c\n", &x, &y, &flag)
        if n < 2 {
            println(n)
            return
        }
        if flag == 'f' || flag == 'F' {
            err = board.Flag(x, y)
            if err != nil {
                fmt.Println(err)
                return
            }
        }else{
            result, err := board.MakeMove(x, y)
            if err != nil {
                fmt.Println(err)
                return
            }
            if result.Result == MineBlown{
                println("BOOM")
                return
            }
        }
    }
    
    // board.PrintRevaled()
}
