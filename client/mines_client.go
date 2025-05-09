package client

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"strconv"
	"sync"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/tomasstrnad1997/mines/mines"
	"github.com/tomasstrnad1997/mines/protocol"
)

type BoardView struct {
    board *[][]rune
    params mines.GameParams
}



type AppState int

const (
    ConnectMenu AppState = iota
	BrowserMenu
    GameStartMenu
    GameScreen
)

var (
    boardMutex sync.Mutex
)

type Menu struct {
    ipEditor widget.Editor
    connectButton widget.Clickable
    connecting bool
    gameEndResult protocol.GameEndType
    
    widthEditor widget.Editor
    heightEditor widget.Editor
    minesEditor widget.Editor
    startButton widget.Clickable

    restartButton widget.Clickable
    newGameButton widget.Clickable

    state AppState

	browser *GameBrowserMenu

}

type Cell struct {
	isMine     bool
	isRevealed bool
	isFlagged  bool
	neighborMines int
    x int
    y int
}


type GameManager struct {
    grid [][]Cell
    cellColorGrid [][]int
    params mines.GameParams
    gameController *protocol.ConnectionController
	matchmakingController *protocol.ConnectionController
}

const (
    cellSpacing int = 2 
)

type pressedMouseButton byte
const (
    NoButton pressedMouseButton = iota
    PrimaryButton
    SecondaryButton
)

func (manager *GameManager) HandleGamemodeUpdateInfo(info mines.GamemodeUpdateInfo) error {
	gameModeId := info.GetGameModeId()
	switch gameModeId{
	case mines.ModeCoop:
		i, ok := info.(*mines.CoopInfoUpdate)
		if !ok {
			return fmt.Errorf("Failed to cast to CoopInfoUpdate")
		}
		if err := manager.ApplyCoopUpdateInfo(i); err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unknown GameId: %d", gameModeId)
	}
	return nil
}

func (manager *GameManager) ApplyCoopUpdateInfo(info *mines.CoopInfoUpdate) error {
	for _, cellInfo := range info.MarksChange {
		manager.cellColorGrid[cellInfo.X][cellInfo.Y] = cellInfo.PlayerId
	}
	return nil
}

func createCell(cell_size int, manager *GameManager, cell *Cell, ops *op.Ops, q input.Source, th *material.Theme, gtx layout.Context ) {
    size := image.Point{X:cell_size, Y:cell_size }
    r := image.Rectangle{Max: size}
    offset := image.Point{X: (cellSpacing+cell_size)*cell.x, Y: (cellSpacing+cell_size)*cell.y}
    defer op.Offset(offset).Push(ops).Pop()
    defer clip.Rect(r).Push(ops).Pop()
    event.Op(ops, cell)
    err := handleCellPressed(ReadCellPresses(cell, q), cell, manager)
    if err != nil {
        println("Failed to send button press:", err.Error())
    }
    c, mark := getCellColorAndMark(cell)
     
    paint.ColorOp{Color: c}.Add(ops)
    paint.PaintOp{}.Add(ops)
	overlayColor := getOverlayColor(cell, manager)
    paint.ColorOp{Color: overlayColor}.Add(ops)
    paint.PaintOp{}.Add(ops)
    drawMark(mark, ops, th, gtx)
}

func getOverlayColor(cell *Cell, manager *GameManager) color.NRGBA{
	playerId := manager.cellColorGrid[cell.x][cell.y]
	switch playerId{
	case 1:
		return color.NRGBA{R: 0xAA, G: 0x00, B: 0x00, A: 0x40} 
	case 2:
		return color.NRGBA{R: 0x00, G: 0xAA, B: 0x00, A: 0x40} 
	default:
		return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x00} 
	}

}

func handleCellPressed(buttonPressed pressedMouseButton, cell *Cell, manager *GameManager) error {
    var mType mines.MoveType
    switch buttonPressed {
    case NoButton:
        return nil
    case PrimaryButton:
        mType = mines.Reveal
    case SecondaryButton:
        mType = mines.Flag
    default:
        return fmt.Errorf("Unknown button pressed")
    }
    encoded, err := protocol.EncodeMove(mines.Move{X: cell.x , Y: cell.y, Type: mType})
    if err != nil {
        return err
    }

    err = manager.gameController.SendMessage(encoded)
    if err != nil {
        return err
    }
    return nil
}

func ReadCellPresses(cell *Cell, q input.Source) pressedMouseButton {
	for {
		ev, ok := q.Event(pointer.Filter{
			Target: cell,
			Kinds:  pointer.Press | pointer.Release,
		})
		if !ok {
			break
		}
        if x, ok := ev.(pointer.Event); ok {
            if x.Kind == pointer.Press {
                if x.Buttons.Contain(pointer.ButtonPrimary) {
                    return PrimaryButton
                }else if x.Buttons.Contain(pointer.ButtonSecondary) {
                    return SecondaryButton
                }
            }
        }
	}
    return NoButton
}

func getCellColorAndMark(cell *Cell) (c color.NRGBA, mark string) {
    mark = ""
    c = color.NRGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xFF} 
    if cell.isRevealed {
        c = color.NRGBA{R: 0xAA, G: 0xAA, B: 0xAA, A: 0xFF} 
        if cell.neighborMines > 0 {
            mark = strconv.Itoa(cell.neighborMines)
        }
        if cell.isMine {
            mark = "X"
        }
    }
    if cell.isFlagged{
        c = color.NRGBA{R: 0xAA, G: 0x00, B: 0x00, A: 0xFF} 
    }
    return c, mark
}

func drawMark(mark string, ops *op.Ops, th *material.Theme, gtx layout.Context) {
    cell_size := int(gtx.Metric.PxPerDp*25)
    offset := image.Point{X:cell_size/4, Y:cell_size/8}
    defer op.Offset(offset).Push(ops).Pop()
    material.Label(th, unit.Sp(18), mark).Layout(gtx)
    
}


func drawConfigMenu(gtx layout.Context, th *material.Theme, menu *Menu) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:    layout.Vertical,
			Spacing: layout.SpaceAround,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Editor(th, &menu.widthEditor, "Width").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx) // Add spacing
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Editor(th, &menu.heightEditor, "Height").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx) // Add spacing
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Editor(th, &menu.minesEditor, "Number of Mines").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx) // Add spacing
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Button(th, &menu.startButton, "Start").Layout(gtx)
			}),
		)
	})
}

func drawBoard(manager *GameManager, ops *op.Ops, q input.Source, th *material.Theme, gtx layout.Context) layout.Dimensions{
    cellSize := int(gtx.Metric.PxPerDp * 25)
    totalWidth := manager.params.Width*cellSize + (manager.params.Width-1)*cellSpacing
    totalHeight := manager.params.Height*cellSize + (manager.params.Height-1)*cellSpacing
    offset := image.Point{X: 10, Y: 10}
    defer op.Offset(offset).Push(ops).Pop()
    boardMutex.Lock()
    for col := range manager.params.Width {
        for row := range manager.params.Height {
            createCell(cellSize, manager, &manager.grid[col][row], ops, q, th, gtx)
        }
    }
    boardMutex.Unlock()
    return layout.Dimensions{
        Size: image.Point{X: totalWidth + offset.X*2, Y: totalHeight + offset.Y*2},
    }   
}

func drawGameScreen(manager *GameManager, ops *op.Ops, q input.Source, th *material.Theme, gtx layout.Context) layout.Dimensions{
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:    layout.Vertical,
			Spacing: layout.SpaceAround,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                return drawBoard(manager, ops, q, th, gtx)
			}),
		)
	})
}

func drawEndGame(gtx layout.Context, th *material.Theme, menu *Menu) layout.Dimensions{
    var txt string
    switch menu.gameEndResult {
    case protocol.Aborted:
        txt = "Aborted"
    case protocol.Win:
        txt = "Game won"
    case protocol.Loss:
        txt = "Game lost"
    default:
        return layout.Dimensions{}
    }
    return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
        return layout.Flex{
            Axis:    layout.Vertical,
            Spacing: layout.SpaceAround,
            Alignment: layout.Middle,
        }.Layout(gtx,
        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
            return material.Label(th, unit.Sp(100), txt).Layout(gtx)
        }),
        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
            return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
        }),
        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
            return material.Button(th, &menu.restartButton, "Restart").Layout(gtx)
        }),
        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
            return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx)
        }),
        layout.Rigid(func(gtx layout.Context) layout.Dimensions {
            return material.Button(th, &menu.newGameButton, "New game").Layout(gtx)
        }),
    )
})
}
func (manager *GameManager) connectToGameServer(w *app.Window, menu *Menu, host string, port uint16){
	fmt.Printf("Connecting to %s:%d\n", host, port)
    go func() {
        menu.connecting = true
        err := manager.gameController.Connect(host, port)
        if err != nil {
            println(err.Error())
        }else{
            menu.state = GameStartMenu
            go func() {
                err := manager.gameController.ReadServerResponse()
                if err != nil {
                    println(err.Error())
                }
                menu.state = ConnectMenu
                w.Invalidate()
            }()
        }
        w.Invalidate()
        menu.connecting = false
    }()
}

func handleConnectButton(w *app.Window, menu *Menu, manager *GameManager){
	manager.connectToGameServer(w, menu, menu.ipEditor.Text(), 42069)
}

func handleStartGameButton(menu *Menu, manager *GameManager){
    width, errw := strconv.Atoi(menu.widthEditor.Text())
    height, errh := strconv.Atoi(menu.heightEditor.Text())
    nMines, errm := strconv.Atoi(menu.minesEditor.Text())
    if errw != nil || errh != nil || errm != nil {
    }else {
        encoded, err := protocol.EncodeGameStart(mines.GameParams{Width: width, Height: height, Mines: nMines, GameMode: mines.ModeCoop})
        if err != nil {
            println(err.Error())
        }else{
    		err = manager.gameController.SendMessage(encoded)
			if err != nil {
				println(err.Error())
			}
        }
    }
}

func initializeGrid(manager *GameManager) {
    boardMutex.Lock()
	manager.cellColorGrid = make([][]int, manager.params.Width)
    manager.grid = make([][]Cell, manager.params.Width)
    for i := range manager.params.Width {
        manager.grid[i] = make([]Cell, manager.params.Height)
        manager.cellColorGrid[i] = make([]int, manager.params.Height)
        for j := range manager.params.Height {
            manager.grid[i][j].x = i
            manager.grid[i][j].y = j
        }
    }
    boardMutex.Unlock()

}

func RegisterMMHandlers(w *app.Window, manager *GameManager, menu *Menu, controller *protocol.ConnectionController){
    controller.RegisterHandler(protocol.SendGameServers, func(bytes []byte) error { 
		infos, err := protocol.DecodeSendGameServers(bytes, nil)
		if err != nil {
			return err
		}
		rows := make([]*GameServerRow, len(infos))
		for i, info := range infos {
			rows[i] = &GameServerRow{info: info}
		}
		menu.browser.servers = rows
		return nil
    })
    controller.RegisterHandler(protocol.ServerSpawned, func(bytes []byte) error { 
		info, err := protocol.DecodeServerSpawned(bytes, nil)
		if err != nil {
			return err
		}
		menu.browser.servers = append(menu.browser.servers, &GameServerRow{info: info})
		return nil
    })
}

func RegisterGUIHandlers(w *app.Window, manager *GameManager, menu *Menu, controller *protocol.ConnectionController){
    controller.RegisterHandler(protocol.GameEnd, func(bytes []byte) error { 
        endType, err := protocol.DecodeGameEnd(bytes)
        if err != nil {
            return err
        }
        menu.gameEndResult = endType
        return nil
    })
    controller.RegisterHandler(protocol.GamemodeInfo, func(bytes []byte) error { 
        info, err := protocol.DecodeGamemodeInfo(bytes)
        if err != nil{
            return err
        }
		if err = manager.HandleGamemodeUpdateInfo(info); err != nil {
			return err
		}
        return nil     
    })
    controller.RegisterHandler(protocol.TextMessage, func(bytes []byte) error { 
        msg, err := protocol.DecodeTextMessage(bytes)
        if err != nil{
            return err
        }
        println(msg)
        return nil     
    })
    controller.RegisterHandler(protocol.StartGame, func(bytes []byte) error { 
        params, err := protocol.DecodeGameStart(bytes)
        if err != nil{
            return err
        }
        manager.params = *params
        initializeGrid(manager)
        menu.state = GameScreen
        menu.gameEndResult = 0
        w.Invalidate()
        return nil     
    })
    controller.RegisterHandler(protocol.CellUpdate, func(bytes []byte) error { 
        updates, err := protocol.DecodeCellUpdates(bytes)
        if err != nil{
            return err
        }
        for _, cell := range updates {
            c := &manager.grid[cell.X][cell.Y]
            if (cell.Value & 0xF0) == 0{
                c.neighborMines = int(cell.Value)
                c.isRevealed = true
                continue
            }
            switch cell.Value {
            case mines.Unflag:
                c.isFlagged = false
            case mines.ShowFlag:
                c.isFlagged = true
            case mines.ShowMine:
                c.isMine = true
                c.isRevealed = true
                
            }
        }
        w.Invalidate()
        return nil
    })
}

func handleRestartButton(manager *GameManager) {
    encoded, err := protocol.EncodeGameStart(manager.params)
    if err != nil {
        println(err.Error())
    }else{
        err = manager.gameController.SendMessage(encoded)
		if err != nil {
			println(err.Error())
		}
    }
}

func handleNewGameButton(menu *Menu) {
    menu.state = GameStartMenu
}

func handleBrowserConnectButton(w *app.Window, menu *Menu, manager *GameManager, server *GameServerRow){
	manager.connectToGameServer(w, menu, server.info.Host, server.info.Port)
}

func (manager *GameManager) refreshServers() error {
	encoded, err := protocol.EncodeGetGameServers(nil)
	if err != nil {
		return err
	}
	err = manager.matchmakingController.SendMessage(encoded)
	if err != nil {
		return err
	}
	return nil
}

func (manager *GameManager) spawnServer(name string) error {
	encoded, err := protocol.EncodeSpawnServerRequest(name, nil)
	if err != nil {
		return err
	}
	if err = manager.matchmakingController.SendMessage(encoded);err != nil {
		return err
	}
	return nil
}


func handleMenuButtons(gtx layout.Context, w *app.Window, menu *Menu, manager *GameManager) {
	if menu.connectButton.Clicked(gtx){
		handleConnectButton(w, menu, manager)
	}
	if menu.startButton.Clicked(gtx) {
		handleStartGameButton(menu, manager)
	}
	if menu.restartButton.Clicked(gtx){
		handleRestartButton(manager)
	}
	if menu.newGameButton.Clicked(gtx){
		handleNewGameButton(menu)
	}

	for _, server := range menu.browser.servers {
		if server.ConnectButton.Clicked(gtx){
			handleBrowserConnectButton(w, menu, manager, server)
		}
	}

	if menu.browser.refreshButton.Clicked(gtx) {
		manager.refreshServers()
	}
	if menu.browser.spawnButton.Clicked(gtx) {
		manager.spawnServer(menu.browser.serverName.Text())
	}
}

func mainLoop(w *app.Window, th *material.Theme, menu *Menu) error {
        var ops op.Ops
        manager := &GameManager{
			gameController: protocol.CreateConnectionController(),
			matchmakingController: protocol.CreateConnectionController(),
		}
		RegisterMMHandlers(w, manager, menu, manager.matchmakingController)
        RegisterGUIHandlers(w, manager, menu, manager.gameController)

		manager.matchmakingController.AttemptReconnect = true
		manager.matchmakingController.Connect("localhost", 42071)
		go manager.matchmakingController.ReadServerResponse()
		
        for {
            switch windowEvent := w.Event().(type){
            case app.FrameEvent:
                gtx := app.NewContext(&ops, windowEvent)
				handleMenuButtons(gtx, w, menu, manager)
                switch menu.state {
                case ConnectMenu:
                    drawBrowserMenu(gtx, th, menu)
                case GameStartMenu:
                    drawConfigMenu(gtx, th, menu)
                case GameScreen:
                    drawGameScreen(manager, &ops, windowEvent.Source, th, gtx)
                    drawEndGame(gtx, th, menu)
                }
                windowEvent.Frame(gtx.Ops)
            case app.DestroyEvent:
                return windowEvent.Err
            }
        }
}


func RunClient() {
    go func() {
        w := new(app.Window)
        w.Option(app.Title("PogySweeper"))
        th := material.NewTheme()
		var servers = []*GameServerRow{}
		browser := &GameBrowserMenu{
			servers: servers,
			list : layout.List{Axis: layout.Vertical},
			}
        menu := &Menu{
            state: ConnectMenu,
			browser: browser,
        }
        // menu.ipEditor.SetText("127.0.0.1")
        menu.ipEditor.SingleLine = true
        menu.widthEditor.SetText("10")
        menu.widthEditor.SingleLine = true
        menu.heightEditor.SetText("20")
        menu.heightEditor.SingleLine = true
        menu.minesEditor.SetText("9")
        menu.minesEditor.SingleLine = true

        err := mainLoop(w, th, menu)
        if err != nil {
            print(err.Error())
        }
        os.Exit(0)
    }()
    app.Main()
}
