package main

import "C"
import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

type Engine struct{}

func (e *Engine) accept(ctx context.Context, conn net.Conn, writeCh chan <- string, readCh <- chan instrument) {
	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	go handleConn(conn, writeCh chan string, readCh chan instrument)
}

func handleConn(conn net.Conn, writeCh chan <- instrument, readCh <- chan string) {
	defer conn.Close()
	for {
		select {
		case msg := <-ch:
			fmt.Printf("Received message %d\n", msg)
			// Do something with the message
		default:
			// If there is no message to read, sleep for a bit
			time.Sleep(100 * time.Millisecond)
		}
		in, err := readInput(conn)
		if err != nil {
			if err != io.EOF {
				_, _ = fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			}
			return
		}
		switch in.orderType {
		case inputCancel:
			fmt.Fprintf(os.Stderr, "Got cancel ID: %v\n", in.orderId)
			outputOrderDeleted(in, true, GetCurrentTimestamp())
		default:
			fmt.Fprintf(os.Stderr, "Got order: %c %v x %v @ %v ID: %v\n",
				in.orderType, in.instrument, in.count, in.price, in.orderId)
			outputOrderAdded(in, GetCurrentTimestamp())
		}
		outputOrderExecuted(123, 124, 1, 2000, 10, GetCurrentTimestamp())
	}
}

func GetCurrentTimestamp() int64 {
	return time.Now().UnixNano()
}
