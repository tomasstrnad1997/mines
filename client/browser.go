package client

import (
	"fmt"
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/tomasstrnad1997/mines/protocol"
)

type GameBrowserMenu struct {
	SpawServerButton widget.Clickable
	servers []*GameServerRow
	list layout.List 

}

type GameServerRow struct {
	info protocol.GameServerInfo
	ConnectButton widget.Clickable	

}


// Couldnt make it work just discard it
func drawHeader(gtx layout.Context, th *material.Theme) layout.Dimensions {
    return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(16), Right: unit.Dp(16), Bottom: unit.Dp(8)}.Layout(gtx,
        func(gtx layout.Context) layout.Dimensions {
            ops := gtx.Ops
			macro := op.Record(ops)
			dims := layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
                    Alignment: layout.Middle,
                }.Layout(gtx,
                    layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                        return material.Body1(th, "Server").Layout(gtx)
                    }),
                    layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
                    layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                        return material.Body1(th,"Players").Layout(gtx)
                    }),
                    layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
                        return layout.Spacer{Width: unit.Dp(0)}.Layout(gtx)
                    }),
                )
			})
			call := macro.Stop()
            rr := clip.RRect{
                Rect: image.Rect(0, 0, gtx.Constraints.Max.X, dims.Size.Y),
                SE: 8,
                SW: 8,
                NE: 8,
                NW: 8,
            }
            paint.FillShape(ops, color.NRGBA{R: 200, G: 200, B: 200, A: 255}, rr.Op(ops))
			call.Add(ops)
			return dims
        })
}

func drawServerRow(gtx layout.Context, th *material.Theme, server *GameServerRow) layout.Dimensions {
    return layout.Inset{Top: unit.Dp(8), Left: unit.Dp(16), Right: unit.Dp(16), Bottom: unit.Dp(8)}.Layout(gtx,
        func(gtx layout.Context) layout.Dimensions {
            ops := gtx.Ops
			macro := op.Record(ops)

			dims := layout.UniformInset(unit.Dp(16)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
                return layout.Flex{
                    Alignment: layout.Middle,
                }.Layout(gtx,
                    layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                        return material.Body1(th, server.info.Name).Layout(gtx)
                    }),
                    layout.Rigid(layout.Spacer{Width: unit.Dp(16)}.Layout),
                    layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                        return material.Body1(th,
                            fmt.Sprintf("%d", server.info.PlayerCount)).Layout(gtx)
                    }),
                    layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
                        return layout.Spacer{Width: unit.Dp(0)}.Layout(gtx)
                    }),
                    layout.Rigid(func(gtx layout.Context) layout.Dimensions {
                        btn := material.Button(th, &server.ConnectButton, "Connect")
                        return btn.Layout(gtx)
                    }),
                )
            })
			call := macro.Stop()
            rr := clip.RRect{
                Rect: image.Rect(0, 0, gtx.Constraints.Max.X, dims.Size.Y),
                SE: 8,
                SW: 8,
                NE: 8,
                NW: 8,
            }
            paint.FillShape(ops, color.NRGBA{R: 240, G: 240, B: 240, A: 255}, rr.Op(ops))
			call.Add(ops)
			return dims
        })
}

func drawBrowserMenu(gtx layout.Context, th *material.Theme, menu *Menu){
	layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return drawHeader(gtx, th)
	}),
		layout.Flexed(1, func(ftx layout.Context) layout.Dimensions {
			return menu.browser.list.Layout(gtx, len(menu.browser.servers), func(gtx layout.Context, i int) layout.Dimensions {
				return drawServerRow(gtx, th, menu.browser.servers[i])
			})
		}),
	)
}

