package main

import (
	"context"
	"embed"
	"log"
	"time"

	"github.com/liyufei916/agent-harbor/internal/contracts"
	"github.com/liyufei916/agent-harbor/internal/desktop"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	application.RegisterEvent[contracts.Event](contracts.HarborEventName)
}

func main() {
	appCtx, cancelApp := context.WithCancel(context.Background())
	defer cancelApp()

	host := desktop.NewApp()
	if err := host.AttachOrchestrator(appCtx); err != nil {
		log.Fatal(err)
	}
	app := application.New(desktopOptions(host, cancelApp))
	newDesktopWindow(app)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// desktopOptions 由正式入口和原生桌面冒烟测试共用。
func desktopOptions(host *desktop.App, cancelApp context.CancelFunc) application.Options {
	return application.Options{
		Name:        "Agent Harbor",
		Description: "密钥留在本机 · Agent 隔离运行",
		Services: []application.Service{
			application.NewService(desktop.NewDesktopAPI(host)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		OnShutdown: func() {
			cancelApp()
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			if err := host.Shutdown(ctx); err != nil {
				log.Printf("应用退出清理失败：%v", err)
			}
		},
	}
}

func newDesktopWindow(app *application.App) *application.WebviewWindow {
	return app.Window.NewWithOptions(application.WebviewWindowOptions{
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
}
