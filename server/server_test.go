package server

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const floatDelta = float64(0.0000000001)

func TestPositionZero(t *testing.T) {
	point := position(time.Second * 0)
	assert.InDelta(t, 1.0, point.X, floatDelta)
	assert.InDelta(t, 0.0, point.Y, floatDelta)
}

func TestPositionHalfPi(t *testing.T) {
	point := position(time.Second / 4)
	assert.InDelta(t, 0.0, point.X, floatDelta)
	assert.InDelta(t, 1.0, point.Y, floatDelta)
}

func TestPositionTwoPi(t *testing.T) {
	point := position(time.Second)
	assert.InDelta(t, 1.0, point.X, floatDelta)
	assert.InDelta(t, 0.0, point.Y, floatDelta)
}
