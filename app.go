package main

import (
	"context"

	"changeme/pkg/chrome"
	"changeme/pkg/config"
	"changeme/pkg/llog"

	"github.com/alphadose/haxmap"
)

// App struct
type App struct {
	controlSignal chan struct{}

	cfg *config.Config
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.cfg = config.Init()

	llog.Init(a.cfg)
}

func (a *App) OpenBrowser() {
	if a.controlSignal == nil {
		a.controlSignal = make(chan struct{}, 1)
	}
	hm := haxmap.New[string, *chrome.RequestInfo]()
	go chrome.RunChromedp(a.ctx, a.cfg, a.controlSignal, hm)
}

func (a *App) CloseBrowser() {
	if len(a.controlSignal) == 0 {
		a.controlSignal <- struct{}{}
	}
}

func (a *App) CloseAllBrowser() {
	close(a.controlSignal)
}

func (a *App) SetConfig(cfg *config.Config) {
	a.cfg = cfg
}
