package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"strings"
	"time"
)

// Revolutions per second
const RPS = float64(1.0)

// Point in Euclidian space
type point struct {
	X, Y float64
}

// Return the current position in the unit circle
func position(d time.Duration) point {
	radians := (d.Seconds() / RPS) * 2.0 * math.Pi
	return point{
		X: math.Cos(radians),
		Y: math.Sin(radians),
	}
}

func emit(cxt context.Context, conn *net.UDPConn) {
	s := time.Now()
	t := time.NewTicker(time.Millisecond * time.Duration(7))
	for {
		select {
		case <-t.C:
			p := position(time.Since(s))
			b, err := json.Marshal(p)
			if err != nil {
				log.Fatal(err)
			}
			_, err = conn.Write(b)
			if err != nil {
				log.Fatal(err)
			}
		case <-cxt.Done():
			t.Stop()
			conn.Close()
			return
		}
	}
}

func serve(c net.Conn) {
	log.Println("New TCP connection, saying hello")
	_, err := c.Write([]byte("HELLO\n"))
	if err != nil {
		log.Println(err)
		return
	}

	cxt := context.Background()
	cancels := make([]context.CancelFunc, 0)
	rw := bufio.NewReadWriter(bufio.NewReader(c), bufio.NewWriter(c))

	for {
		cmd, err := rw.ReadString('\n')
		if err != nil {
			log.Fatal(err)
			return
		}
		switch strings.TrimSpace(cmd) {
		case "start":
			raddr, err := net.ResolveUDPAddr("udp", "localhost:5001")
			if err != nil {
				log.Fatal(err)
			}

			log.Println("Starting point transmission via UDP")
			u, err := net.DialUDP("udp", nil, raddr)
			if err != nil {
				log.Println(err)
				return
			}
			child, cancel := context.WithCancel(cxt)
			cancels = append(cancels, cancel)
			go emit(child, u)

			_, err = c.Write([]byte("OK\n"))
			if err != nil {
				log.Println(err)
				return
			}
		case "stop":
			log.Println("Stopping point transmission")
			for _, cancel := range cancels {
				cancel()
			}
			_, err = c.Write([]byte("OK\n"))
			if err != nil {
				log.Println(err)
				return
			}
		case "bye":
			log.Println("Acknowledging goodbye")
			_, err = c.Write([]byte("OK\n"))
			if err != nil {
				log.Println(err)
				return
			}
			c.Close()
			return
		default:
			_, err = fmt.Fprintf(rw, "Unknown command: %s.\n", cmd)
			rw.Flush()
			if err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func Start() {
	n, err := net.Listen("tcp", "localhost:5000")
	if err != nil {
		log.Fatal(err)
	}

	for {
		c, err := n.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go serve(c)
	}
}
