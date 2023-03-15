package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type InstrumentChannel struct {
    instrumentName string
    channel chan inputPackage
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

	instrumentChMap := make(map[string] chan inputPackage)
	clientWriteChSlice := []chan InstrumentChannel{}
	clientReadCh := make(chan string, 500)
	newClientCh := make(chan chan InstrumentChannel, 45)
	go func() {
		for {
			select {
			case instrument := <-clientReadCh:
				_, ok := instrumentChMap[instrument]
				if (!ok) {
					instrumentCh := make(chan inputPackage, 200)
					go readChannel(instrumentCh)
					instrumentChMap[instrument] = instrumentCh
					for _, client := range clientWriteChSlice {
						client <- InstrumentChannel{instrument, instrumentCh}
					}
				}
			case newWriteCh := <- newClientCh:
				clientWriteChSlice = append(clientWriteChSlice, newWriteCh)
				for instrument, instrumentCh := range instrumentChMap {
					newWriteCh <- InstrumentChannel{instrument, instrumentCh}
				}
			default:
				time.Sleep(time.Millisecond)
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
