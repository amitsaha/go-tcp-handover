package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

var originalWD, _ = os.Getwd()
var tcpListener *net.TCPListener
var fileListener net.Listener

func useInheritedSockets(listenFdCount string) {

	// Here we only have one supported socket
	_, err := strconv.Atoi(listenFdCount)
	if err != nil {
		log.Printf("Returning. Error: %v\n", err)
		return
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
	// we don't close lFile since that's the file that carries
	// our listener over to the next process if there is one
	for {
		conn, err := fileListener.Accept()
		if err != nil {
			log.Fatal("Accept error for fileListener: ", err)
		}
		log.Printf("Serving request from: PPID:%d PID:%d.\n", os.Getppid(), os.Getpid())

		go func(c net.Conn) {
			io.Copy(c, c)
			c.Close()
		}(conn)
	}
}

func main() {

	var err error
	var parentDone = make(chan bool)

	// Register handler for SIGUSR1 for gracefully exiting a process and starting a new one
	// like for binary upgrades
	cP := make(chan os.Signal, 1)
	signal.Notify(cP, syscall.SIGUSR1)

	go func() {
		s := <-cP
		log.Printf("Got signal:%v. Starting new process.\n", s)
		allFiles := []*os.File{os.Stdin, os.Stdout, os.Stderr}

		if tcpListener != nil {
			file, err := tcpListener.File()
			if err != nil {
				log.Fatal("Error getting listener file", err)
			}
			allFiles = append(allFiles, file)
		}

		var env []string
		for _, v := range os.Environ() {
			env = append(env, v)
		}
		if len(os.Getenv("LISTEN_FDS")) == 0 {
			env = append(env, fmt.Sprintf("LISTEN_FDS=1"))
		}

		p := os.ProcAttr{
			Dir:   originalWD,
			Env:   env,
			Files: allFiles,
		}

		_, err = os.StartProcess(os.Args[0], os.Args, &p)
		if err != nil {
			log.Printf("Error starting new process: %v\n", err)
		} else {
			log.Println("Signaling exit to:", os.Getpid())
			parentDone <- true
		}
	}()

	// exit loop for the process handing over to the new process
	go func() {
		<-parentDone
		log.Printf("PPID:%d PID:%d. Exiting now.\n", os.Getppid(), os.Getpid())
		//FIXME drain the existing connections
		if tcpListener != nil {
			tcpListener.Close()
		}
		if fileListener != nil {
			fileListener.Close()
		}
		os.Exit(1)
	}()

	log.Printf("PPID:%d PID:%d. Starting now.\n", os.Getppid(), os.Getpid())

	if len(os.Getenv("LISTEN_FDS")) != 0 {
		log.Println("LISTEN_FDS detected")
		useInheritedSockets(os.Getenv("LISTEN_FDS"))
	}

	lAddr := net.TCPAddr{
		Port: 2000,
	}
	tcpListener, err = net.ListenTCP("tcp", &lAddr)
	if err != nil {
		log.Fatal(err)
	}
	for {
		// FIXME: exit this loop after tcpListener.Close() is called
		conn, err := tcpListener.Accept()
		if err != nil {
			log.Fatal("Accept error for tcpListener: ", err)
		}
		log.Printf("Serving request from: PPID:%d PID:%d.\n", os.Getppid(), os.Getpid())
		go func(c net.Conn) {
			io.Copy(c, c)
			c.Close()
		}(conn)
	}
}
