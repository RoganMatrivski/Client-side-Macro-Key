package main

import (
	"math"
	"math/bits"
	"reflect"
	"strings"
)

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

func valueToBar(value byte, barLength int, barString string) string {
	mappedValue := int(math.Floor((float64(value) / float64(100)) * float64(barLength)))
	bar := strings.Repeat(barString, mappedValue)
	empty := strings.Repeat(" ", barLength-mappedValue)
	return bar + empty
}
