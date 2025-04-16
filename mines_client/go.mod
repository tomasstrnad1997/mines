module github.com/tomasstrnad1997/mines_client

go 1.23.4

replace github.com/tomasstrnad1997/mines => ../mines

replace github.com/tomasstrnad1997/mines_protocol => ../mines_protocol

require (
	github.com/tomasstrnad1997/mines v0.0.0-00010101000000-000000000000
	github.com/tomasstrnad1997/mines_protocol v0.0.0-00010101000000-000000000000
)

require (
	gioui.org v0.8.0 // indirect
	gioui.org/cpu v0.0.0-20210817075930-8d6a761490d2 // indirect
	gioui.org/shader v1.0.8 // indirect
	github.com/go-text/typesetting v0.2.1 // indirect
	golang.org/x/exp v0.0.0-20240707233637-46b078467d37 // indirect
	golang.org/x/exp/shiny v0.0.0-20240707233637-46b078467d37 // indirect
	golang.org/x/image v0.18.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
)
