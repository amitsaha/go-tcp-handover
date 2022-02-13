package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	var parentDone = make(chan bool)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Hi there, i am a server")
	})

	srv := http.Server{
		Handler: mux,
	}
	l := getNetworkListener()
	// Register handler for SIGUSR1 for gracefully exiting a process and starting a new one
	// like for binary upgrades
	cP := make(chan os.Signal, 1)
	signal.Notify(cP, syscall.SIGUSR1)

	go func() {
		s := <-cP
		log.Printf("Got signal:%v. Starting new process.\n", s)
		allFiles := []*os.File{os.Stdin, os.Stdout, os.Stderr}

		if lTCP, ok := l.(*net.TCPListener); ok {
			file, err := lTCP.File()
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

		_, err := os.StartProcess(os.Args[0], os.Args, &p)
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
		srv.Shutdown(context.Background())
	}()

	log.Fatal(srv.Serve(l))
	// FIXME: wait for shutdown
}
