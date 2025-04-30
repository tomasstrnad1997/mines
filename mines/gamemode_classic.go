package mines

type Classic struct{}

func (c *Classic) Init(board *Board) {}

func (c *Classic) Name() string {
	return "Classic"
}

func (c *Classic) GameModeId() GameModeId {
	return ModeClassic
}

func (c *Classic) OnMove(b *Board, move Move, result *MoveResult) (GamemodeUpdateInfo, error) {
	return nil, nil
}

