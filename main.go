package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

type InstrumentChannel struct {
    instrumentName string
    channel chan input
}

func handleSigs(cancel func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	cancel()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <socket path>\n", os.Args[0])
		return
	}

	socketPath := os.Args[1]
	if err := os.RemoveAll(socketPath); err != nil {
		log.Fatal("remove existing sock error: ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		handleSigs(cancel)
	}()

	l, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal("listen error: ", err)
	}
	go func() {
		<-ctx.Done()
		if err := l.Close(); err != nil {
			log.Fatal("close listener error: ", err)
		}
	}()

	instrumentChMap := make(map[string] chan input)
	clientWriteChSlice := []chan instrument{}
	clientReadCh := make(chan string)
	newClientCh := make(chan chan instrument)

	go func() {
		for {
			select {
			case instrument := <-clientReadCh:
				instrumentCh := make(chan input)
				//go makeInstrument(instrumentCh)
				instrumentChMap[instrument] = instrumentCh
				for _, client := range clientWriteChSlice {
					client <- InstrumentChannel{instrument, instrumentCh}
				}
				
			case newWriteCh := <- newClientCh:
				clientWriteChSlice = clientWriteChSlice.append(clientWriteChSlice, newWriteCh)
				
				for instrument, instrumentCh := range instrumentChMap {
					newWriteCh <- InstrumentChannel{instrument, instrumentCh}
				}
			}
		}
	}()

	var e Engine
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal("accept error: ", err)
		}

		e.accept(ctx, conn, clientReadCh, newClientCh)
	}
}
