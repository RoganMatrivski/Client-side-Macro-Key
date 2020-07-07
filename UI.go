package main

import (
	"io/ioutil"
	"os"
	"strings"

	"fyne.io/fyne"
	"fyne.io/fyne/app"
	"fyne.io/fyne/layout"
	"fyne.io/fyne/widget"
)

var (
	appUI       fyne.App
	mainWindow  fyne.Window
	aboutWindow fyne.Window
	logWindow   fyne.Window

	selectedProfileLabel  *widget.Label
	profileListLabels     []*tappableLabel
	profileListWithScroll *widget.ScrollContainer

	volumeBar *widget.ProgressBar

	logTexts           *widget.Label
	logScrollContainer *widget.ScrollContainer

	logLabelArray []*widget.Label
)

func switchSelectedProfile(idx int) {
	currentProfileIndex = idx

}

func setupUI() {
	appUI = app.New()

	selectedProfileLabel = widget.NewLabel("Selected Profile:\n")

	profileListLabels = make([]*tappableLabel, len(conf.Profiles))
	for i, profile := range conf.Profiles {
		newProfileLabel := newTappableLabel(profile, func(l *tappableLabel) {
			switchProfileToIndex(l.Profiles.Index)
		})
		profileListLabels[i] = newProfileLabel
	}

	profileListObjects := make([]fyne.CanvasObject, len(conf.Profiles))
	for i, e := range profileListLabels {
		profileListObjects[i] = e
	}

	profileListWithScroll = widget.NewVScrollContainer(widget.NewVBox(profileListObjects...))
	profileListWithScroll.SetMinSize(fyne.NewSize(300, 280))

	volumeBar = widget.NewProgressBar()
	volumeBar.Max = 100
	volumeBar.Min = 0
	groupedVolumeBar := widget.NewGroup("Volume", volumeBar)

	aboutButton := widget.NewButton("About", showAboutWindow)
	logButton := widget.NewButton("Logs", showLogWindow)
	buttonGroup := widget.NewHBox(aboutButton, logButton)

	footerGroup := widget.NewVBox(groupedVolumeBar, buttonGroup)

	// TODO: Change the vbox to border layout after adding the buttons
	container := fyne.NewContainerWithLayout(layout.NewBorderLayout(selectedProfileLabel, footerGroup, nil, nil), selectedProfileLabel, footerGroup, profileListWithScroll)

	mainWindow = appUI.NewWindow("Main")
	mainWindow.SetContent(container)

	// ==========================================================================

	logTexts = widget.NewLabel("")
	logScrollContainer = widget.NewScrollContainer(logTexts)
	logScrollContainer.SetMinSize(fyne.NewSize(500, 300))

	// Add the existing data to the log window
	viewedLogs := logs[len(logs)-(40+1):]

	logTexts.SetText(strings.Join(viewedLogs, "\n"))
	logScrollContainer.Offset = fyne.NewPos(0, logScrollContainer.Content.Size().Height-logScrollContainer.Size().Height)
	logTexts.Refresh()
	logScrollContainer.Refresh()

	// mainWindow.Resize(fyne.NewSize(300, 500))

	mainWindow.CenterOnScreen()

	mainWindow.SetOnClosed(func() {
		os.Exit(0)
	})
}

func showLogWindow() {
	exitLogWindow := widget.NewButton("Exit", func() {
		logWindow.Close()
	})

	logWindowContent := fyne.NewContainerWithLayout(layout.NewBorderLayout(nil, exitLogWindow, nil, nil), exitLogWindow, logScrollContainer)

	logWindow = appUI.NewWindow("Log")
	logWindow.SetContent(logWindowContent)
	logWindow.Resize(fyne.NewSize(300, 500))

	logWindow.CenterOnScreen()

	logWindow.Show()
}

func showAboutWindow() {
	exitAboutButton := widget.NewButton("Exit", func() {
		aboutWindow.Hide()
	})

	aboutTextSrc, _ := ioutil.ReadFile("./about.txt")
	aboutText := widget.NewLabel(string(aboutTextSrc))
	aboutText.Wrapping = fyne.TextWrapWord
	aboutTextScroll := widget.NewVScrollContainer(aboutText)

	aboutContainer := fyne.NewContainerWithLayout(layout.NewBorderLayout(nil, exitAboutButton, nil, nil), aboutTextScroll, exitAboutButton)

	aboutWindow = appUI.NewWindow("About")
	aboutWindow.SetContent(aboutContainer)
	aboutWindow.Resize(fyne.NewSize(600, 500))
	aboutWindow.SetFixedSize(true)

	aboutWindow.SetOnClosed(func() {
		aboutWindow.Close()
	})

	aboutWindow.CenterOnScreen()

	aboutWindow.Show()
}
