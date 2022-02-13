package main

import (
	"log"
	"net"
	"os"
	"strconv"
)

var originalWD, _ = os.Getwd()
var tcpListener *net.TCPListener
var fileListener net.Listener

func useInheritedSockets(listenFdCount string) net.Listener {

	// Here we only have one supported socket
	_, err := strconv.Atoi(listenFdCount)
	if err != nil {
		log.Printf("Returning. Error: %v\n", err)
		return nil
	}

	// hardcoding it to 3
	// listener file descriptor is actually not accessible, the one that
	// we have access to is the file descriptor corresponding to the socket which is
	// different and hence useless
	lFile := os.NewFile(uintptr(3), "listener")
	if lFile == nil {
		log.Fatal("Error creating os.File")
	}
	fileListener, err = net.FileListener(lFile)
	if err != nil {
		log.Fatalf("Error creating listener from file: %v", err)
	}
	return fileListener
}

func getNetworkListener() net.Listener {

	var err error

	if len(os.Getenv("LISTEN_FDS")) != 0 {
		log.Println("LISTEN_FDS detected")
		return useInheritedSockets(os.Getenv("LISTEN_FDS"))
	}

	lAddr := net.TCPAddr{
		Port: 2000,
	}
	tcpListener, err = net.ListenTCP("tcp", &lAddr)
	if err != nil {
		log.Fatal(err)
	}
	return tcpListener
}
