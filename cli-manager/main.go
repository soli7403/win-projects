// TermGo — Copyright (c) 2026 kai. BrightTerm v1.1.0(MIT) 기반 Go + Wails 포팅.
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var g_assets embed.FS

func main() {
	pstrApp, err := NewApp()
	if err != nil {
		log.Fatal(err)
	}
	opts := &options.App{
		Title:         "TermGo",
		MinWidth:      900,
		MinHeight:     560,
		AssetServer:   &assetserver.Options{Assets: g_assets},
		OnStartup:     pstrApp.startup,
		OnDomReady:    pstrApp.restoreWindowPosition,
		OnBeforeClose: pstrApp.beforeClose,
		OnShutdown:    pstrApp.shutdown,
		Bind:          []interface{}{pstrApp},
	}
	pstrApp.applyWindowOptions(opts)
	err = wails.Run(opts)
	if err != nil {
		log.Fatal(err)
	}
}
