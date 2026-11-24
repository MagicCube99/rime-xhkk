package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	state := NewAppState("")

	mygo.App.WhenReady(func() {
		var win *mygo.Window
		win = mygo.NewWindow(mygo.WindowOptions{
			Title:           "rime-xhkk 词条速加",
			Width:           490,
			Height:          204,
			UseContentSize:  true,
			DisableMaximize: true,
			DisableResize:   true,
			Content: ui.View(func(c *ui.Context) {
				BuildUI(c, state, win)
			}),
		})
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
