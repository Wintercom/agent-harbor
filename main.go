package main

import (
	"context"
	"embed"
	"log"
	"time"

	"github.com/Wintercom/agent-harbor/internal/contracts"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	application.RegisterEvent[contracts.Event](contracts.HarborEventName)
}

func main() {
	host := NewApp()
	api := NewDesktopAPI(host)

	// 研发 A 就绪后在此注入：
	// WireSessionService(host, orchestrator.New)
	// host.SetAgentDetector(...)

	app := application.New(application.Options{
		Name:        "Agent Harbor",
		Description: "密钥留在本机 · Agent 隔离运行",
		Services: []application.Service{
			application.NewService(api),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "Agent Harbor",
		Width:  1100,
		Height: 720,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
			TitleBar: application.MacTitleBarDefault,
		},
		BackgroundColour: application.NewRGB(24, 26, 32),
		URL:              "/",
	})

	app.OnShutdown(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		if err := host.shutdown(ctx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
