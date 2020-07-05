package main

import "time"

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
	ShortPressAction string `json:"shortPressAction"`
	LongPressAction  string `json:"longPressAction"`
}

type buttonState struct {
	buttonIndex        int
	previousSignal     int
	lastDebounceTime   time.Time
	lastPressedTime    time.Time
	buttonPressedTime  time.Time
	buttonReleasedTime time.Time
	associatedButton   Buttons
}

type loggingLevel int
type profileSwitchType int

func (lv loggingLevel) String() string { return [...]string{"ERROR", "WARNING", "INFO"}[lv] }
