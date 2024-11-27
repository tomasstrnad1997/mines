package main

import (
	"fmt"
	"math/rand"
)

type Cell struct
{
    mine bool
    revealed bool

}

type Board struct {
    height int
    width int
    cells [][]Cell

}

type InvalidBoardParamsError struct {
    height int
    width int
    mines int
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
    cells := make([][]Cell, width)
    for i := range cells {
        cells[i] = make([]Cell, height)
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

func main() {
    board, err := CreateBoard(5, 5, 5);
    if err != nil {
        fmt.Println(err)
        return
    }
    
    fmt.Println(board)
}
