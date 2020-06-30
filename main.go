package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/bits"
	"os/exec"
	"reflect"
	"time"

	"github.com/itchyny/volume-go"
	"github.com/jacobsa/go-serial/serial"
)

const (
	buttonDebounceTime    time.Duration = time.Millisecond * 50
	buttonLongPressTime   time.Duration = time.Millisecond * 300
	potentiometerDeadzone byte          = 1
)

type Configs struct {
	AHKExecutablePath string    `json:"AHKExecutablePath"`
	Buttons           []Buttons `json:"buttons"`
}

type Buttons struct {
	ShortPressAction   string `json:"shortPressAction"`
	LongPressAction    string `json:"longPressAction"`
	previousSignal     int
	lastDebounceTime   time.Time
	lastPressedTime    time.Time
	buttonPressedTime  time.Time
	buttonReleasedTime time.Time
}

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

var (
	conf         Configs
	buttonStates []Buttons
	signature    = [4]byte{2, 4, 3, 4}
)

func setup() {
	jsonFile, _ := ioutil.ReadFile("./configuration.json")
	err := json.Unmarshal(jsonFile, &conf)
	if err != nil {
		panic(err)
	}

	buttonStates = conf.Buttons

	if len(buttonStates) > 8 {
		panic("More than eight keys macro are not supported by hardware")
	}

	for _, b := range buttonStates {
		if b.ShortPressAction == "" && b.LongPressAction == "" {
			panic("One of the key have both of the actions unassigned.")
		}
	}
}

func byteAbs(b byte) byte {
	if b < 0 {
		return b
	}
	return -b
}

var potentiometerState, prevPotentiometerState byte

func potentiometerLoop() {
	for {
		if byteAbs(potentiometerState-prevPotentiometerState) > potentiometerDeadzone {
			err := volume.SetVolume(int(potentiometerState))
			if err != nil {
				log.Fatalf("set volume failed: %+v", err)
			}

			fmt.Printf("set volume to %v%%\n", potentiometerState)
			prevPotentiometerState = potentiometerState
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
			panic(err)
		}
	} else {
		// Longpress
		cmd := exec.Command(conf.AHKExecutablePath, b.LongPressAction)
		err := cmd.Start()
		if err != nil {
			panic(err)
		}
	}

	// fmt.Printf("%v\n%v\n%v\n%v\n", b.buttonPressedTime, b.buttonReleasedTime, t1, t2)
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

func main() {
	setup()

	go potentiometerLoop()

	// Set up options.
	options := serial.OpenOptions{
		PortName:        "COM3",
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

	signature := []byte{2, 4, 3, 4}
	for {
		// Find offset
		buff := make([]byte, 6)
		port.Read(buff)
		extBuff := append(buff, buff...)

		var offset int
		for i := range extBuff {
			if i > len(extBuff)-4 {
				break
			}

			// fmt.Print(extBuff[i:i+4], i)
			if arrayCompare(extBuff[i:i+4], signature) {
				offset = i
				break
			}
		}

		serialData := extBuff[offset+4 : offset+6]

		potentiometerData := serialData[1]
		buttonsData := byteToBits(serialData[0])
		reverseAny(buttonsData)

		potentiometerState = potentiometerData
		// buttonState = buttonsData

		processButtonSignal(buttonsData)

		// fmt.Println(buttonsData, potentiometerData)
	}
}
