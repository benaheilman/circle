package client

import (
	"bufio"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/StephaneBunel/bresenham"
)

const scale = 1024

type point struct {
	X, Y float64
}

type circle struct {
	Image  draw.Image
	Scale  int
	Border int
}

func newCircle() circle {
	c := circle{Image: nil, Scale: scale, Border: scale / 16}
	c.Image = image.NewRGBA(image.Rect(0, 0, c.Scale+c.Border*2, c.Scale+c.Border*2))
	draw.Draw(c.Image, c.Image.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	return c
}

func (c *circle) Plotter(x, y int, color color.Color) {
	c.Image.Set(x, y, color)
}

func (c *circle) line(p1, p2 point) {
	trans := func(v float64) int {
		return int((v+1.0)*float64(c.Scale)/2.0) + c.Border
	}
	bresenham.DrawLine(
		c.Image, trans(p1.X), trans(p1.Y), trans(p2.X), trans(p2.Y), color.Black,
	)
}

func (c *circle) save(name string) error {
	file, err := os.Create(name)
	if err != nil {
		return err
	}
	defer file.Close()

	err = png.Encode(file, c.Image)
	if err != nil {
		return err
	}
	return nil
}

func receive(cxt context.Context, c *net.UDPConn, ch chan<- point) {
	buf := make([]byte, 1024)
	for {
		c.SetReadDeadline(time.Now().Add(time.Second * time.Duration(30)))
		n, _, err := c.ReadFromUDP(buf)
		if err != nil {
			log.Fatalf("%T: %v", err, err)
		}
		if n > 0 {
			p := point{}
			err := json.Unmarshal(buf[0:n], &p)
			if err != nil {
				log.Fatal(err)
			}
			ch <- p
		}

		select {
		case <-cxt.Done():
			close(ch)
			return
		default:
		}
	}
}

func sketch(cxt context.Context, wg *sync.WaitGroup, c *net.UDPConn, name string) {
	defer wg.Done()

	circle := newCircle()
	defer circle.save(name)

	ch := make(chan point)
	go receive(cxt, c, ch)

	var last *point = nil
	for {
		select {
		case p, ok := <-ch:
			if !ok {
				log.Println("Channel closed")
				return
			}
			if last != nil {
				circle.line(*last, p)
			}
			last = &p
		case <-cxt.Done():
			return
		}
	}
}

func Start(output string) {
	log.Println("Starting TCP connection with server")
	c, err := net.Dial("tcp", "localhost:5000")
	if err != nil {
		log.Fatal(err)
	}
	rw := bufio.NewReadWriter(bufio.NewReader(c), bufio.NewWriter(c))
	resp, err := rw.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	if r := strings.TrimSpace(resp); r != "HELLO" {
		log.Fatalf("Server greeted me with: %s", r)
	}

	addr, err := net.ResolveUDPAddr("udp", "localhost:5001")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening for UDP packets")
	u, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	cxt := context.Background()
	child, cancel := context.WithCancel(cxt)
	go sketch(child, &wg, u, output)

	log.Println("Telling server to start transmissing points")
	_, err = c.Write([]byte("start\n"))
	if err != nil {
		log.Fatal(err)
	}
	resp, err = rw.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	if r := strings.TrimSpace(resp); r != "OK" {
		log.Fatalf("Server greeted me with: %s", r)
	}

	time.Sleep(time.Second * time.Duration(10))
	cancel()
	wg.Wait()

	log.Println("Telling server to stop transmissing points")
	_, err = c.Write([]byte("stop\n"))
	if err != nil {
		log.Fatal(err)
	}
	resp, err = rw.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	if r := strings.TrimSpace(resp); r != "OK" {
		log.Fatalf("Server greeted me with: %s", r)
	}

	log.Println("Saying goodbye to server")
	_, err = c.Write([]byte("bye\n"))
	if err != nil {
		log.Fatal(err)
	}
	resp, err = rw.ReadString('\n')
	if err != nil {
		log.Fatal(err)
	}
	if r := strings.TrimSpace(resp); r != "OK" {
		log.Fatalf("Server greeted me with: %s", r)
	}
}
