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

func createBoard(params mines.GameParams) *BoardView {
    board := make([][]rune, params.Width)
    for i := range board {
        board[i] = make([]rune, params.Height)
        for j := 0; j < params.Height; j++{
            board[i][j] = '#'
        }
    }
    return &BoardView{&board, params}
}

func (board *BoardView) Print(){
    print("X")
    for i:=0; i < 10; i++{
        print(i % 10)
    }
    println()
    for y := 0; y < board.params.Height; y++{
        print(y % 10)
        for x := 0; x < board.params.Width; x++{
            fmt.Printf("%c",(*board.board)[x][y])
        }
        println()
    }
}

func RegisterHandlers(){
    var board *BoardView 
    RegisterHandler(protocol.GameEnd, func(bytes []byte) error { 
        endType, err := protocol.DecodeGameEnd(bytes)
        if err != nil {
            return err
        }
        switch endType {
        case protocol.Win:
            println("Game won")
        case protocol.Loss:
            println("Game lost")
        case protocol.Aborted:
            println("Game aborted")
        }
        board = nil
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
        board = createBoard(*params)
        board.Print()
        return nil     
    })
    RegisterHandler(protocol.CellUpdate, func(bytes []byte) error { 
        updates, err := protocol.DecodeCellUpdates(bytes)
        if err != nil{
            return err
        }
        for _, cell := range updates {
            var rep rune 
            if (cell.Value & 0xF0) == 0{
                rep = rune(cell.Value) + '0'
            }else if cell.Value == mines.ShowFlag{
                rep = 'F'
            }else if cell.Value == mines.ShowMine{
                rep = 'X'
            }else if cell.Value == mines.Unflag{
                rep = '#'
            }else {
                rep = '?'
            }
            (*board.board)[cell.X][cell.Y] = rep
            // fmt.Printf("Move: %d, %d - %c\n", cell.X, cell.Y, rep)
        }
        board.Print()
        return nil
    })

}

func runTerminalClient() {
    client, err := createClient("localhost:8080")

    if err != nil {
        println(err.Error())
        os.Exit(1)
    }
    RegisterHandlers()
    go ReadServerResponse(client)

    for {
        var x, y int
        var flag rune
        flag = 'X'
        n, _ := fmt.Scanf("%d %d %c\n", &x, &y, &flag)
        var move mines.Move
        if n < 2 {
            fmt.Println("Incorrect input")
            continue
        }
        if flag == 'f' || flag == 'F' {
            move = mines.Move{X: x, Y: y, Type: mines.Flag}
            
        }else if flag == 'G' {
            encoded, err := protocol.EncodeGameStart(mines.GameParams{Width: x, Height: x, Mines: y})
            if err != nil {
                println(err.Error())
                continue
            }
            client.Write(encoded)
            continue
        }else{
            move = mines.Move{X: x, Y: y, Type: mines.Reveal}
        }
        encoded, err := protocol.EncodeMove(move)
        if err != nil {
            println(err.Error())
            continue
        }
        client.Write(encoded)
    }   
}

type AppState int

const (
    ConnectMenu AppState = iota
    GameStartMenu
    GameScreen
)

type Menu struct {
    ipEditor widget.Editor
    connectButton widget.Clickable
    connecting bool
    
    widthEditor widget.Editor
    heightEditor widget.Editor
    minesEditor widget.Editor
    startButton widget.Clickable

    state AppState

}

type CutomButton struct {
    
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


func createCell(manager *GameManager, cell *Cell, ops *op.Ops, q input.Source, th *material.Theme, gtx layout.Context ) {
    cell_size := 75
    size := image.Point{X:cell_size, Y:cell_size }
    r := image.Rectangle{Max: size}
    offset := image.Point{X: (2+cell_size)*cell.x, Y: (2+cell_size)*cell.y}
    defer op.Offset(offset).Push(ops).Pop()
    defer clip.Rect(r).Push(ops).Pop()
    event.Op(ops, cell)
    buttonPressed := 0
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
                    buttonPressed = 1
                }else if x.Buttons.Contain(pointer.ButtonSecondary) {
                    buttonPressed = 2
                }
            }
        }
	}

    var c color.NRGBA
    if buttonPressed > 0 {
        var mType mines.MoveType
        if buttonPressed == 1 {
            mType = mines.Reveal
        } else {
            mType = mines.Flag
        }
        encoded, err := protocol.EncodeMove(mines.Move{X: cell.x , Y: cell.y, Type: mType})
        if err != nil {
            println(err.Error())
            return
        }
        manager.server.Write(encoded)
    }
    
    mark := ""
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
    paint.ColorOp{Color: c}.Add(ops)
    paint.PaintOp{}.Add(ops)
    drawMark(mark, ops, th, gtx)
    
}
func drawMark(mark string, ops *op.Ops, th *material.Theme, gtx layout.Context) {
    offset := image.Point{X:23, Y:10}
    defer op.Offset(offset).Push(ops).Pop()
    material.Label(th, unit.Sp(25), mark).Layout(gtx)
}

func drawConnectMenu(gtx layout.Context, th *material.Theme, menu *Menu) layout.Dimensions {
	return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{
			Axis:    layout.Vertical,
			Spacing: layout.SpaceAround,
		}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Editor(th, &menu.ipEditor, "Enter IP Address").Layout(gtx)
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
				return material.Editor(th, &menu.widthEditor, "Enter Width").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx) // Add spacing
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Editor(th, &menu.heightEditor, "Enter Height").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Spacer{Height: unit.Dp(8)}.Layout(gtx) // Add spacing
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Editor(th, &menu.minesEditor, "Enter Number of Mines").Layout(gtx)
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

func drawGame(manager *GameManager, ops *op.Ops, q input.Source, th *material.Theme, gtx layout.Context){
    for row := 0; row < manager.params.Height; row++ {
        for col := 0; col < manager.params.Width; col++ {
            createCell(manager, &manager.grid[row][col], ops, q, th, gtx)
        }
    }
    
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

func intializeGrid(manager *GameManager) {
    manager.grid = make([][]Cell, manager.params.Height)
    for i := 0; i < manager.params.Height; i++ {
        manager.grid[i] = make([]Cell, manager.params.Width)
        for j := 0; j < manager.params.Width; j++ {
            manager.grid[i][j].x = j
            manager.grid[i][j].y = i
        }
    }

}

func RegisterGUIHandlers(w *app.Window, manager *GameManager, menu *Menu){
    RegisterHandler(protocol.GameEnd, func(bytes []byte) error { 
        endType, err := protocol.DecodeGameEnd(bytes)
        if err != nil {
            return err
        }
        switch endType {
        case protocol.Win:
            println("Game won")
        case protocol.Loss:
            println("Game lost")
        case protocol.Aborted:
            println("Game aborted")
        }
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
        intializeGrid(manager)
        menu.state = GameScreen
        w.Invalidate()
        return nil     
    })
    RegisterHandler(protocol.CellUpdate, func(bytes []byte) error { 
        updates, err := protocol.DecodeCellUpdates(bytes)
        if err != nil{
            return err
        }
        for _, cell := range updates {
            c := &manager.grid[cell.Y][cell.X]
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

func draw(w *app.Window, th *material.Theme, menu *Menu) error {
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
                switch menu.state {
                case ConnectMenu:
                    drawConnectMenu(gtx, th, menu)
                case GameStartMenu:
                    drawConfigMenu(gtx, th, menu)
                case GameScreen:
                    drawGame(manager, &ops, windowEvent.Source, th, gtx)
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
        w.Option(app.Title("Minesweeper"))
        th := material.NewTheme()

        menu := &Menu{
            state: ConnectMenu,
        }
        // menu.ipEditor.SetText("127.0.0.1:42069")
        menu.ipEditor.SingleLine = true
        menu.widthEditor.SetText("10")
        menu.widthEditor.SingleLine = true
        menu.heightEditor.SetText("10")
        menu.heightEditor.SingleLine = true
        menu.minesEditor.SetText("10")
        menu.minesEditor.SingleLine = true

        err := draw(w, th, menu)
        if err != nil {
            print(err.Error())
        }
        os.Exit(0)
    }()
    app.Main()
}
