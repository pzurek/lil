package main

import (
	_ "embed" // Add this import for the embed directive
	"log"

	"github.com/getlantern/systray"
)

//go:embed assets/icon_template_36.png
var iconData []byte

func main() {
	// systray.Run requires the main thread on macOS, so start it in a goroutine
	// if other setup is needed, but here we run it directly.
	systray.Run(onReady, onExit)
}

func onReady() {
	log.Println("Lil systray app starting...")
	systray.SetTemplateIcon(iconData, iconData) // Pass icon data for both light and dark modes
	systray.SetTitle("")
	systray.SetTooltip("Linear Ticket Lister")

	// Add a Quit menu item
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// Goroutine to handle menu item clicks
	go func() {
		for {
			select {
			case <-mQuit.ClickedCh:
				log.Println("Quit item clicked")
				systray.Quit()
				return // Exit the goroutine when Quit is clicked
			}
		}
	}()

	log.Println("Systray ready.")
	// In the future, we'll add the API call and dynamic menu items here
}

func onExit() {
	// Clean up resources if needed
	log.Println("Lil systray app finished.")
}
