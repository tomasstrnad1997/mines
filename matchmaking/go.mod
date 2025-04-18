module github.com/tomasstrnad1997/mines_matchmaking

go 1.24.2

replace github.com/tomasstrnad1997/mines_server => ../mines_server

replace github.com/tomasstrnad1997/mines => ../mines

replace github.com/tomasstrnad1997/mines_protocol => ../mines_protocol

require github.com/tomasstrnad1997/mines_server v0.0.0-00010101000000-000000000000

require (
	github.com/tomasstrnad1997/mines v0.0.0-00010101000000-000000000000 // indirect
	github.com/tomasstrnad1997/mines_protocol v0.0.0-00010101000000-000000000000 // indirect
)
