package players

type Player struct {
	ID uint32
	Name string
	PasswordHash string
}

type PlayerStore interface {
	CreatePlayer(username, hash string) error
	FindPlayerByName(username string) (*Player, error)
}
