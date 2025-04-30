package mines



type Coop struct{
	boardPlayerMarks [][]int
	playerScores map[int] int
}

type PlayerMarkChange struct {
	X int
	Y int
	PlayerId int
}

type CoopInfoUpdate struct {
	MarksChange []PlayerMarkChange
	PlayerScores map[int] int
}

func (c *CoopInfoUpdate) GetGameModeId() GameModeId{
	return ModeCoop
}

func (c *Coop) Init(board *Board) {
    grid := make([][]int, board.Width)
    for i := range grid {
        grid[i] = make([]int, board.Height)
    }
	c.boardPlayerMarks = grid
	c.playerScores = make(map[int]int)

}

func (c *Coop) Name() string {
	return "Coop"
}

func (c *Coop) GameModeId() GameModeId {
	return ModeCoop
}

func (c *Coop) OnMove(b *Board, move Move, result *MoveResult) (GamemodeUpdateInfo, error) {
	if result.Result == NoChange {
		return nil, nil
	}
	var updates []PlayerMarkChange
	for _, res := range result.UpdatedCells {
		if res.Flagged || res.Revealed {
			c.boardPlayerMarks[res.X][res.Y] = move.PlayerId
			c.playerScores[move.PlayerId]++
			updates = append(updates, PlayerMarkChange{res.X, res.Y, move.PlayerId})
		}
		if !res.Flagged && !res.Revealed { // Unflag
			c.boardPlayerMarks[res.X][res.Y] = 0
			c.playerScores[move.PlayerId]--
			updates = append(updates, PlayerMarkChange{res.X, res.Y, 0})
		}
	}
	info := &CoopInfoUpdate{MarksChange: updates, PlayerScores: c.playerScores}
	return info, nil
}

