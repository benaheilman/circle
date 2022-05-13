package client

import "testing"

func TestLine(t *testing.T) {
	c := newCircle()
	c.line(point{0.0, 0.0}, point{1.0, 1.0})
	c.save("test.png")
}
