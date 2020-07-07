package main

import (
	"fyne.io/fyne"
	"fyne.io/fyne/widget"
)

type tappableLabel struct {
	widget.Label
	Profiles
	// Add the struct you want to extend here

	onTapped func(*tappableLabel)
}

func newTappableLabel(profile Profiles, tapped func(*tappableLabel)) *tappableLabel {
	label := &tappableLabel{Profiles: profile, onTapped: tapped}
	label.ExtendBaseWidget(label)
	label.SetText(profile.Name)

	return label
}

func (l *tappableLabel) Tapped(_ *fyne.PointEvent) {
	if l.onTapped != nil {
		l.onTapped(l)
	}
}

func (l *tappableLabel) TappedSecondary(_ *fyne.PointEvent) {
}
