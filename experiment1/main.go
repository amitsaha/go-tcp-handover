package main

import (
	"io"
	"log"
	"net"
	"os"
	"os/signal"
)

func main() {
	var nFileListener, nNetListener int
	lAddr := net.TCPAddr{
		Port: 2000,
	}
	l, err := net.ListenTCP("tcp", &lAddr)
	if err != nil {
		log.Fatal(err)
	}
	file, err := l.File()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Listener file descriptor: %v\n", file.Fd())
	defer l.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		s := <-c
		log.Printf("Got signal:%v\n", s)
		log.Printf("Requests served by network listener: %d\n", nNetListener)
		log.Printf("Requests served by file listener: %d\n", nFileListener)

		os.Exit(1)
	}()

	go func() {
		lFile := os.NewFile(file.Fd(), "listener")
		if lFile == nil {
			log.Fatal("Error creating os.File")
		}
		defer lFile.Close()
		listenerFile, err := net.FileListener(lFile)
		if err != nil {
			log.Fatalf("Error creating listener from file: %v", err)
		}
		defer listenerFile.Close()
		for {
			conn, err := listenerFile.Accept()
			if err != nil {
				log.Fatal(err)
			}

			nFileListener += 1
			go func(c net.Conn) {
				io.Copy(c, c)
				c.Close()
			}(conn)
		}

	}()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		nNetListener += 1
		go func(c net.Conn) {
			io.Copy(c, c)
			c.Close()
		}(conn)
	}
}
