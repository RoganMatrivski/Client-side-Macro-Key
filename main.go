package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/itchyny/volume-go"
	"github.com/jacobsa/go-serial/serial"
	"github.com/jroimartin/gocui"
)

const (
	logERROR loggingLevel = iota
	logWARNING
	logINFO
	nextProfile profileSwitchType = iota
	prevProfile
	buttonDebounceTime    time.Duration = time.Millisecond * 50
	buttonLongPressTime   time.Duration = time.Millisecond * 300
	potentiometerDeadzone byte          = 1
	logPath               string        = "./applog.log"
)

var (
	conf Configs

	potentiometerData      byte
	prevPotentiometerState byte

	buttonStates        []buttonState
	profileButtonStates []buttonState
	currentProfileIndex int

	signature     = []byte{2, 4, 3, 4}
	offset    int = -1

	logs          []string
	infoLogger    *log.Logger
	warningLogger *log.Logger
	errorLogger   *log.Logger

	g *gocui.Gui
)

func arrayLog(s string) {
	logs = append(logs[1:], s)
}

func logger(s string, lv loggingLevel) {
	switch lv {
	case logERROR:
		errorLogger.Println(s)
	case logWARNING:
		warningLogger.Println(s)
	case logINFO:
		infoLogger.Println(s)
	}

	arrayLog(s)

	g.Update(func(g *gocui.Gui) error {
		v, err := g.View("log")
		_, maxY := v.Size()
		if err != nil {
			panic(err)
		}
		v.Clear()
		fmt.Fprintln(v, strings.Join(logs[len(logs)-maxY:], "\n"))
		return nil
	})
}

func setup() {
	logs = make([]string, 100)
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	infoLogger = log.New(file, "INFO\t: ", log.Ldate|log.Ltime|log.Lshortfile)
	warningLogger = log.New(file, "WARNING\t: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger = log.New(file, "ERROR\t: ", log.Ldate|log.Ltime|log.Lshortfile)

	jsonFile, _ := ioutil.ReadFile("./configuration.json")
	if err := json.Unmarshal(jsonFile, &conf); err != nil {
		logger(err.Error(), logERROR)
		os.Exit(1)
	}

	for _, profile := range conf.Profiles {
		if len(profile.Buttons) > conf.InputNumber {
			logger(fmt.Sprintf("More than %v keys macro are not supported by hardware", conf.InputNumber), logERROR)
		}

		for _, b := range profile.Buttons {
			if b.ShortPressAction == "" && b.LongPressAction == "" {
				logger("One of the key have both of the actions unassigned.", logWARNING)
			}
		}
	}

	updateProfileInterface()

	buttonStates = make([]buttonState, conf.InputNumber)

	// Assign default profile
	for i, button := range conf.Profiles[0].Buttons {
		buttonStates[i].associatedButton = button

		// Add button index for logging reasons
		buttonStates[i].buttonIndex = i
	}

	profileButtonStates = make([]buttonState, 2)

	// Assigning the profile switching action here. I use the existing button structs because i'm lazy. And i know this is will bite me in the future.
	profileButtonStates[0].associatedButton.ShortPressAction = "PREV"
	profileButtonStates[1].associatedButton.ShortPressAction = "NEXT"
}

func potentiometerLoop() {
	// Idk if i should use for loop + sleep combo here, but it works tho.
	for {
		if byteAbs(potentiometerData-prevPotentiometerState) > potentiometerDeadzone {
			err := volume.SetVolume(int(potentiometerData))
			if err != nil {
				logger(err.Error(), logERROR)
			}

			logger(fmt.Sprintf("set volume to %v%%", potentiometerData), logINFO)

			g.Update(func(g *gocui.Gui) error {
				v, err := g.View("volume")
				maxX, _ := v.Size()
				if err != nil {
					panic(err)
				}
				v.Clear()
				fmt.Fprint(v, " [", valueToBar(potentiometerData, maxX-4, "="), "] ")
				return nil
			})

			prevPotentiometerState = potentiometerData
		}

		time.Sleep((1 * time.Second) / 20)
	}
}

func buttonAction(b buttonState) {
	if b.buttonPressedTime.Add(buttonLongPressTime).After(b.buttonReleasedTime) {
		// Shortpress
		cmd := exec.Command(conf.AHKExecutablePath, b.associatedButton.ShortPressAction)
		err := cmd.Start()
		if err != nil {
			logger(err.Error(), logERROR)
		}
		logger(fmt.Sprintf("Button %v short pressed", b.buttonIndex), logINFO)
	} else {
		// Longpress
		cmd := exec.Command(conf.AHKExecutablePath, b.associatedButton.LongPressAction)
		err := cmd.Start()
		if err != nil {
			logger(err.Error(), logERROR)
		}
		logger(fmt.Sprintf("Button %v long pressed", b.buttonIndex), logINFO)
	}
}

func processButtonSignal(buttonSignal []int) {
	for i := range buttonSignal {
		if i+1 > len(buttonStates) {
			return
		}
		currentButton := &buttonStates[i]

		if buttonSignal[i] == 1 && currentButton.previousSignal == 0 {
			if time.Now().After(currentButton.lastDebounceTime.Add(buttonDebounceTime)) {
				// Button is pressed
				currentTime := time.Now()
				currentButton.lastDebounceTime = currentTime
				currentButton.buttonPressedTime = currentTime
				currentButton.previousSignal = 1
			}
		}

		if buttonSignal[i] == 0 && currentButton.previousSignal == 1 {
			// Button is released
			currentButton.buttonReleasedTime = time.Now()
			currentButton.previousSignal = 0
			buttonAction(*currentButton)
		}
	}
}

func switchProfile(action profileSwitchType) {
	switch action {
	case prevProfile:
		if currentProfileIndex-1 >= 0 {
			currentProfileIndex--
		}
	case nextProfile:
		if currentProfileIndex+1 < len(conf.Profiles) {
			currentProfileIndex++
		}
	}

	// Assign default profile
	for i, button := range conf.Profiles[currentProfileIndex].Buttons {
		buttonStates[i].associatedButton = button

		// Add button index for logging reasons
		buttonStates[i].buttonIndex = i
	}

	updateProfileInterface()
}

func updateProfileInterface() {
	g.Update(func(g *gocui.Gui) error {
		v, err := g.View("profile")
		_, maxY := v.Size()
		maxY--
		if err != nil {
			panic(err)
		}
		v.Clear()

		var profileNames []string
		profileNames = make([]string, maxY)

		if len(conf.Profiles) > maxY {
			switch {
			case currentProfileIndex < maxY/2:
				for i, profile := range conf.Profiles[:maxY] {
					profileNames[i] = profile.Name
				}
			case currentProfileIndex > len(conf.Profiles)-(maxY/2):
				for i, profile := range conf.Profiles[len(conf.Profiles)-maxY:] {
					profileNames[i] = profile.Name
				}
			default:
				for i, profile := range conf.Profiles[currentProfileIndex-maxY/2 : currentProfileIndex+maxY/2] {
					profileNames[i] = profile.Name
				}
			}
		} else {
			for i, profile := range conf.Profiles {
				profileNames[i] = profile.Name
			}
		}

		// Add a selected indicator on the selected profile
		profileNames[currentProfileIndex] = "> " + profileNames[currentProfileIndex]
		fmt.Fprint(v, strings.Join(profileNames, "\n"))
		logger(fmt.Sprint(maxY, maxY/2, len(conf.Profiles)), logINFO)
		return nil
	})
}

func processProfileButtonSignal(buttonSignal []byte) {
	for i := range buttonSignal {
		if i+1 > len(profileButtonStates) {
			return
		}
		currentButton := &profileButtonStates[i]

		if buttonSignal[i] == 1 && currentButton.previousSignal == 0 {
			if time.Now().After(currentButton.lastDebounceTime.Add(buttonDebounceTime)) {
				// Button is pressed
				currentTime := time.Now()
				currentButton.lastDebounceTime = currentTime
				currentButton.buttonPressedTime = currentTime
				currentButton.previousSignal = 1
			}
		}

		if buttonSignal[i] == 0 && currentButton.previousSignal == 1 {
			// Button is released
			currentButton.buttonReleasedTime = time.Now()
			currentButton.previousSignal = 0
			switch currentButton.associatedButton.ShortPressAction {
			case "PREV":
				// Prev button
				logger("Prev button is pressed", logINFO)
				switchProfile(prevProfile)
			case "NEXT":
				// Next button
				logger("Next button is pressed", logINFO)
				switchProfile(nextProfile)
			}
		}
	}
}

func readSerialData(serialPort io.ReadWriteCloser) {
	for {
		// Read the data from serial port
		buff := make([]byte, 8)
		serialPort.Read(buff)

		// Extend it
		extBuff := append(buff, buff...)

		// Find the offset
		// This process will run for each data received, and i know that it's not that efficient.
		// But i guess if this works, why should i fix it?
		for i := range extBuff {
			if i > len(extBuff)-4 {
				break
			}

			// Check it with a predetermined data signature.
			// I chose 2434 for... reasons. Just search it on google. It's an alias for an agency.
			if arrayCompare(extBuff[i:i+4], signature) {
				offset = i
				break
			}
		}

		// Get serial data from extended serial data based from offset
		serialData := extBuff[offset+4 : offset+8]

		// logger(fmt.Sprint(serialData, offset, extBuff), logINFO)

		// Set the potentiometer data to a variable to read later on the potentiometerLoop
		potentiometerData = serialData[1]

		// Process the button data
		buttonsData := byteToBits(serialData[0])
		reverseAny(buttonsData)
		processButtonSignal(buttonsData)

		for i := range serialData[2:] {
			if serialData[2+i] == 1 {
				serialData[2+i] = 0
			} else {
				serialData[2+i] = 1
			}
		}
		processProfileButtonSignal(serialData[2:])
	}
}

func mainLayout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("log", maxX/4, 3, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Logs"
		v.Autoscroll = true
	}

	if v, err := g.SetView("volume", maxX/4, 0, maxX-1, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Audio Volume"
		v.Autoscroll = true
	}

	if v, err := g.SetView("profile", 0, 0, (maxX/4)-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Profiles"
		v.Autoscroll = true
	}

	return nil
}

func main() {
	test()
	os.Exit(0)
	// All of these below is for the interface.
	var err error
	g, err = gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		logger(err.Error(), logERROR)
	}

	// Close GUI after loop ends
	defer g.Close()

	g.SetManagerFunc(mainLayout)

	// Set GUI keybindings
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		return gocui.ErrQuit
	}); err != nil {
		logger(err.Error(), logERROR)
	}
	if err := g.SetKeybinding("log", gocui.KeyArrowUp, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			scrollView(v, -1)
			return nil
		}); err != nil {
		panic(err)
	}
	if err := g.SetKeybinding("log", gocui.KeyArrowDown, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			scrollView(v, 1)
			return nil
		}); err != nil {
		panic(err)
	}

	// =====================================================

	setup()

	// Run the potentiometer loop
	go potentiometerLoop()

	// Set up options.
	options := serial.OpenOptions{
		PortName:        conf.ArduinoPort,
		BaudRate:        19200,
		DataBits:        8,
		StopBits:        1,
		MinimumReadSize: 4,
	}

	// Open the port.
	port, err := serial.Open(options)
	if err != nil {
		log.Fatalf("serial.Open: %v", err)
	}

	// Make sure to close it later.
	defer port.Close()

	go readSerialData(port)

	// Main GUI Loop
	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		logger(err.Error(), logERROR)
	}
}

func scrollView(v *gocui.View, dy int) error {
	if v != nil {
		v.Autoscroll = false
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+dy); err != nil {
			return err
		}
	}
	return nil
}
