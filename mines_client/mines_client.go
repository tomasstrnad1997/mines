package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
	"net"
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

	"github.com/tomasstrnad1997/mines"
	protocol "github.com/tomasstrnad1997/mines_protocol"
)

type BoardView struct {
    board *[][]rune
    params mines.GameParams
}

type MessageHandler func([]byte) error
var messageHandlers = make(map[protocol.MessageType]MessageHandler)
func HandleMessage(bytes []byte) error {

    msgType := protocol.MessageType(bytes[0])
	handler, exists := messageHandlers[msgType]
	if !exists {
		return fmt.Errorf("No handler registered for message type: %d", msgType)
	}
	return handler(bytes)
}

func RegisterHandler(msgType protocol.MessageType, handler MessageHandler) {
	messageHandlers[msgType] = handler
}
func createClient(servAddr string) (*net.TCPConn, error){
    tcpAddr, err := net.ResolveTCPAddr("tcp", servAddr)
    if err != nil {
        println("Reslove tpc failed:")
        return nil, err
    }
    conn, err := net.DialTCP("tcp", nil, tcpAddr)
    // println("RESOLVED TCP")
    if err != nil {
        println("Dial failed:")
        return nil, err
    }
    // println("DIALED TCP")
    return conn, nil
}

func ReadServerResponse(client net.Conn) error{
    reader := bufio.NewReader(client)
    for {
        header := make([]byte, protocol.HeaderLength)
		bytesRead, err := reader.Read(header)
        if err != nil {
            return fmt.Errorf("Lost connection to server\n")
        }
		if bytesRead != protocol.HeaderLength{
            return fmt.Errorf("Failed to read message\n")
		}
        messageLenght := int(binary.BigEndian.Uint32(header[2:protocol.HeaderLength]))
        message := make([]byte, messageLenght+protocol.HeaderLength)
        copy(message[0:protocol.HeaderLength], header)
        _, err = io.ReadFull(reader, message[protocol.HeaderLength:])
        if err != nil {
            return err
        }
        err = HandleMessage(message)    
        if err != nil {
            println(err.Error())
        }
    }
    
}

type AppState int

const (
    ConnectMenu AppState = iota
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
    server *net.TCPConn
    grid [][]Cell
    params mines.GameParams
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
    drawMark(mark, ops, th, gtx)
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
    _, err = manager.server.Write(encoded)
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

func drawConnectMenu(gtx layout.Context, th *material.Theme, menu *Menu) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:    layout.Vertical,
			Spacing: layout.SpaceAround,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Editor(th, &menu.ipEditor, "IP Address").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(16)}.Layout(gtx) // Add spacing
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                btn :=  material.Button(th, &menu.connectButton, "Connect")
                return btn.Layout(gtx) 
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if menu.connecting {
					return material.Label(th, unit.Sp(16), fmt.Sprintf("Connecting to %s", menu.ipEditor.Text())).Layout(gtx)
				}
				return layout.Dimensions{}
			}),
		)
	})
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
    for col := 0; col < manager.params.Width; col++ {
        for row := 0; row < manager.params.Height; row++ {
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

func handleConnectButton(w *app.Window, menu *Menu, manager *GameManager){
    fmt.Printf("Connecting to %s\n", menu.ipEditor.Text())
    go func() {
        menu.connecting = true
        client, err := createClient(menu.ipEditor.Text()+":42069")
        if err != nil {
            println(err.Error())
        }else{
            manager.server = client
            menu.state = GameStartMenu
            go func() {
                err := ReadServerResponse(client)
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

func handleStartGameButton(menu *Menu, manager *GameManager){
    width, errw := strconv.Atoi(menu.widthEditor.Text())
    height, errh := strconv.Atoi(menu.heightEditor.Text())
    nMines, errm := strconv.Atoi(menu.minesEditor.Text())
    if errw != nil || errh != nil || errm != nil {
    }else {
        encoded, err := protocol.EncodeGameStart(mines.GameParams{Width: width, Height: height, Mines: nMines})
        if err != nil {
            println(err.Error())
        }else{
            manager.server.Write(encoded)
        }
    }
}

func initializeGrid(manager *GameManager) {
    boardMutex.Lock()
    manager.grid = make([][]Cell, manager.params.Width)
    for i := 0; i < manager.params.Width; i++ {
        manager.grid[i] = make([]Cell, manager.params.Height)
        for j := 0; j < manager.params.Height; j++ {
            manager.grid[i][j].x = i
            manager.grid[i][j].y = j
        }
    }
    boardMutex.Unlock()

}

func RegisterGUIHandlers(w *app.Window, manager *GameManager, menu *Menu){
    RegisterHandler(protocol.GameEnd, func(bytes []byte) error { 
        endType, err := protocol.DecodeGameEnd(bytes)
        if err != nil {
            return err
        }
        menu.gameEndResult = endType
        return nil
    })
    RegisterHandler(protocol.TextMessage, func(bytes []byte) error { 
        msg, err := protocol.DecodeTextMessage(bytes)
        if err != nil{
            return err
        }
        println(msg)
        return nil     
    })
    RegisterHandler(protocol.StartGame, func(bytes []byte) error { 
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
    RegisterHandler(protocol.CellUpdate, func(bytes []byte) error { 
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
        manager.server.Write(encoded)
    }
}

func handleNEwGameButton(menu *Menu) {
    menu.state = GameStartMenu
}

func mailLoop(w *app.Window, th *material.Theme, menu *Menu) error {
        var ops op.Ops
        manager := &GameManager{}
        RegisterGUIHandlers(w, manager, menu)
        for {
            switch windowEvent := w.Event().(type){
            case app.FrameEvent:
                gtx := app.NewContext(&ops, windowEvent)
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
                    handleNEwGameButton(menu)
                }
                switch menu.state {
                case ConnectMenu:
                    drawConnectMenu(gtx, th, menu)
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

func main() {
    go func() {
        w := new(app.Window)
        w.Option(app.Title("PogySweeper"))
        th := material.NewTheme()

        menu := &Menu{
            state: ConnectMenu,
        }
        // menu.ipEditor.SetText("127.0.0.1")
        menu.ipEditor.SingleLine = true
        menu.widthEditor.SetText("10")
        menu.widthEditor.SingleLine = true
        menu.heightEditor.SetText("20")
        menu.heightEditor.SingleLine = true
        menu.minesEditor.SetText("9")
        menu.minesEditor.SingleLine = true

        err := mailLoop(w, th, menu)
        if err != nil {
            print(err.Error())
        }
        os.Exit(0)
    }()
    app.Main()
}
