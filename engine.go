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

type CommandTuple struct {
    cmd    inputType
    id     uint32
    price  uint32
    count  uint32
	exId   uint32
}

func (e *Engine) accept(ctx context.Context, conn net.Conn, writeCh chan <- string, newClientCh chan <- chan instrument) {
	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	go handleConn(conn, writeCh, newClientCh)
}

func handleConn(conn net.Conn, writeCh chan <- string, newClientCh chan <- chan instrument) {
	defer conn.Close()
	instrumentChMap := make(map[string] chan input)

	readCh := make(chan instrument)
	newClientCh <- readCh

	for {
		select {
		case msg := <-readCh:
			instrumentChMap[msg.instrumentName] = msg.channel
		default:
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
				writeCh := instrumentChMap[in.instrument]
				writeCh <- in
				//outputOrderAdded(in, GetCurrentTimestamp())

			}
			outputOrderExecuted(123, 124, 1, 2000, 10, GetCurrentTimestamp())
			}
	}
}

func GetCurrentTimestamp() int64 {
	return time.Now().UnixNano()
}

func findMatch(cmd inputType, price uint32, count uint32, activeID uint32, tickerSlice []CommandTuple) {
    switch cmd {
		
    case inputBuy: {
		sellPrice := price
		bestIndex := -1
		amt := count

		// Find best match
		for i := 0; i < len(tickerSlice); i++ {
			if (tickerSlide[i].cmd == inputSell && tickerSlice[i].price <= price) {
				sellPrice = tickerSlice[i].price
				bestIndex = i
			}
		}

		// If a match is found...
		if bestIndex != -1 {

			tickerSlice[bestIndex].exId += 1

			// Active order has higher count
			if (amt >= tickerSlice[bestIndex].count) {
				amt = amt - tickerSlice[bestIndex].count
				outputOrderExecuted(tickerSlice[bestIndex].id, activeID, sellPrice, tickerSlide[bestIndex].count, GetCurrentTimestamp())
				// tickerSlice.remove(bestIndex)
			}

			// Resting order has higher count
			else {
				tickerSlice[bestIndex].count -= amt
				outputOrderExecuted(tickerSlice[bestIndex].id, activeID, sellPrice, amt, GetCurrentTimestamp())
				amt = 0
			}
		}

		/*
			If amt = 0, order was settled
			If amt = prev, order found no match
			if amt < prev, order matched with resting and looks again
		*/
		return amt
	}
        
    case inputSell: {
		buyPrice := price
		bestIndex := -1
		amt := count
		// tickerSlice := slice stored in ticker goroutine

		// Find best match
		for i := 0; i < len(tickerSlice); i++ {
			if (tickerSlide[i].cmd == inputBuy && tickerSlice[i].price >= price) {
				buyPrice = tickerSlice[i].price
				bestIndex = i
			}
		}

		// If a match is found...
		if bestIndex != -1 {

			tickerSlice[bestIndex].exId += 1

			// Active order has higher count
			if (amt >= tickerSlice[bestIndex].count) {
				amt = amt - tickerSlice[bestIndex].count
				outputOrderExecuted(tickerSlice[bestIndex].id, activeID, sellPrice, tickerSlide[bestIndex].count, GetCurrentTimestamp())
				// tickerSlice.remove(bestIndex)
			}

			// Resting order has higher count
			else {
				tickerSlice[bestIndex].count -= amt
				outputOrderExecuted(tickerSlice[bestIndex].id, activeID, sellPrice, amt, GetCurrentTimestamp())
				amt = 0
			}
		}

		/*
			If amt = 0, order was settled
			If amt = prev, order found no match
			if amt < prev, order matched with resting and looks again
		*/
		return amt
	}

    default:
        fmt.Println("Invalid command type")
    }
}

func handleOrder(in input, tickerSlice []CommandTuple) {
	cmd := in.orderType
	id := in.orderId
	price := in.price
	instrument := in.instrument
	num := in.count
	for num > 0 {
		prevNum := num
		num = findMatch(cmd, price, num, id, tickerSlice)
		if (num == prevNum) {
			break
		}
	}

	if (num != 0) {
		CommandTuple tup := {cmd, id, price, num, 0}
		tickerSlice = append(tickerSlice, tup)
		outputOrderAdded(in, GetCurrentTimestamp())
	}

}

func readChannel(ch chan input) {
	tickerSlice := []CommandTuple{}
	for {
		select {
			case inputVar := <-ch:
			handleOrder(inputVar, tickerSlice)
	  }
	}
}