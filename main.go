package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/bits"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"github.com/itchyny/volume-go"
	"github.com/jacobsa/go-serial/serial"
	"github.com/jroimartin/gocui"
)

type Configs struct {
	ArduinoPort       string     `json:"ArduinoPort"`
	InputNumber       int        `json:"InputNumber"`
	LogLevel          string     `json:"LogLevel"`
	AHKExecutablePath string     `json:"AHKExecutablePath"`
	Profiles          []Profiles `json:"Profiles"`
}

type Profiles struct {
	Name    string    `json:"Name"`
	Buttons []Buttons `json:"Buttons"`
}

type Buttons struct {
	ShortPressAction   string `json:"shortPressAction"`
	LongPressAction    string `json:"longPressAction"`
	previousSignal     int
	lastDebounceTime   time.Time
	lastPressedTime    time.Time
	buttonPressedTime  time.Time
	buttonReleasedTime time.Time
	buttonID           int
}

type loggingLevel int

const (
	logERROR loggingLevel = iota
	logWARNING
	logINFO
	buttonDebounceTime    time.Duration = time.Millisecond * 50
	buttonLongPressTime   time.Duration = time.Millisecond * 300
	potentiometerDeadzone byte          = 1
	logPath               string        = "./applog.log"
)

func (lv loggingLevel) String() string { return [...]string{"ERROR", "WARNING", "INFO"}[lv] }

func reverseAny(s interface{}) {
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func arrayCompare(a1 []byte, a2 []byte) bool {
	for i, b := range a1 {
		if b != a2[i] {
			return false
		}
	}

	return true
}

func byteToBits(data byte) (st []int) {
	st = make([]int, 8) // Performance x 2 as no append occurs.
	for j := 0; j < 8; j++ {
		if bits.LeadingZeros8(data) == 0 {
			// No leading 0 means that it is a 0
			// Extra author comments: I revert the data because i'm a bit too lazy to revert it on arduino
			st[j] = 0
		} else {
			st[j] = 1
		}
		data = data << 1
	}
	return
}

func byteAbs(b byte) byte {
	if b < 0 {
		return b
	}
	return -b
}

var (
	conf Configs

	potentiometerData      byte
	prevPotentiometerState byte

	buttonStates []Buttons

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
	file, err := os.Create(logPath)
	if err != nil {
		panic(err)
	}

	infoLogger = log.New(file, "INFO\t: ", log.Ldate|log.Ltime|log.Lshortfile)
	warningLogger = log.New(file, "WARNING\t: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger = log.New(file, "ERROR\t: ", log.Ldate|log.Ltime|log.Lshortfile)

	jsonFile, _ := ioutil.ReadFile("./configuration.json")
	if err := json.Unmarshal(jsonFile, &conf); err != nil {
		logger(err.Error(), logERROR)
	}

	buttonStates = conf.Profiles[0].Buttons

	if len(buttonStates) > conf.InputNumber {
		logger("More than eight keys macro are not supported by hardware", logERROR)
	}

	for i, b := range buttonStates {
		if b.ShortPressAction == "" && b.LongPressAction == "" {
			logger("One of the key have both of the actions unassigned.", logWARNING)
		}
		buttonStates[i].buttonID = i
	}
}

func valueToBar(value byte, barLength int, barString string) string {
	mappedValue := int(math.Floor((float64(value) / float64(100)) * float64(barLength)))
	bar := strings.Repeat(barString, mappedValue)
	empty := strings.Repeat(" ", barLength-mappedValue)
	return bar + empty
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

func buttonAction(b Buttons) {
	if b.buttonPressedTime.Add(buttonLongPressTime).After(b.buttonReleasedTime) {
		// Shortpress
		cmd := exec.Command(conf.AHKExecutablePath, b.ShortPressAction)
		err := cmd.Start()
		if err != nil {
			logger(err.Error(), logERROR)
		}
		logger(fmt.Sprintf("Button %v short pressed", b.buttonID), logINFO)
	} else {
		// Longpress
		cmd := exec.Command(conf.AHKExecutablePath, b.LongPressAction)
		err := cmd.Start()
		if err != nil {
			logger(err.Error(), logERROR)
		}
		logger(fmt.Sprintf("Button %v long pressed", b.buttonID), logINFO)
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

func readSerialData(serialPort io.ReadWriteCloser) {
	for {
		// Read the data from serial port
		buff := make([]byte, 6)
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
		serialData := extBuff[offset+4 : offset+6]

		// logger(fmt.Sprint(serialData, offset, extBuff), logINFO)

		// Set the potentiometer data to a variable to read later on the potentiometerLoop
		potentiometerData = serialData[1]

		// Process the button data
		buttonsData := byteToBits(serialData[0])
		reverseAny(buttonsData)
		processButtonSignal(buttonsData)
	}
}

func mainLayout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	if v, err := g.SetView("log", maxX/4, 3, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Autoscroll = true
	}

	if v, err := g.SetView("volume", maxX/4, 0, maxX-1, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Autoscroll = true
	}

	if v, err := g.SetView("profile", 0, 0, (maxX/4)-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Autoscroll = true
	}

	return nil
}

func main() {
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
