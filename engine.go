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

type inputPackage struct {
	in input
	timestamp int64
}

func (e *Engine) accept(ctx context.Context, conn net.Conn, writeCh chan <- string, newClientCh chan <- chan InstrumentChannel) {
	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	go handleConn(conn, writeCh, newClientCh)
}

func handleConn(conn net.Conn, writeCh chan <- string, newClientCh chan <- chan InstrumentChannel) {
	defer conn.Close()
	instrumentChMap := make(map[string] chan inputPackage)
	idMap := make(map[uint32] string)
	readCh := make(chan InstrumentChannel, 500)
	newClientCh <- readCh

	for {
		select {
		case msg := <-readCh:
			instrumentChMap[msg.instrumentName] = msg.channel //3
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
				//outputOrderDeleted(in, true, GetCurrentTimestamp())
				instrument := idMap[in.orderId]
				write, ok := instrumentChMap[instrument]
				for (!ok) { //may need a while here
					newCh := <- readCh
					instrumentChMap[newCh.instrumentName] = newCh.channel
					write, ok = instrumentChMap[instrument]
				}
				write <- inputPackage{in, GetCurrentTimestamp()};
			default:
				idMap[in.orderId] = in.instrument
				fmt.Fprintf(os.Stderr, "default")
				fmt.Fprintf(os.Stderr, "Got order: %c %v x %v @ %v ID: %v\n",
					in.orderType, in.instrument, in.count, in.price, in.orderId)
				write, ok := instrumentChMap[in.instrument] 
				for (!ok) { //may need a while here
					writeCh <- in.instrument //1
					newCh := <- readCh
					instrumentChMap[newCh.instrumentName] = newCh.channel
					write, ok = instrumentChMap[in.instrument]
				}
				write <- inputPackage{in, GetCurrentTimestamp()} //4

			}
			
			}
	}
}

func GetCurrentTimestamp() int64 {
	return time.Now().UnixNano()
}

func findMatch(cmd inputType, price uint32, count uint32, activeID uint32, tickerSlice *[]CommandTuple, ticker string, time int64) uint32 {

    switch cmd {
		
    case 'B': 
		sellPrice := price
		bestIndex := -1
		amt := count

		// Find best match

		for i := len(*tickerSlice) - 1; 0 <= i; i-- {
			if ((*tickerSlice)[i].cmd == 'S' && (*tickerSlice)[i].price <= sellPrice) {
				sellPrice = (*tickerSlice)[i].price
				bestIndex = i
			}
		}

		// If a match is found...
		if bestIndex != -1 {

			(*tickerSlice)[bestIndex].exId += 1

			// Active order has higher count
			if amt >= (*tickerSlice)[bestIndex].count {
				amt = amt - (*tickerSlice)[bestIndex].count
				outputOrderExecuted((*tickerSlice)[bestIndex].id, activeID, (*tickerSlice)[bestIndex].exId, sellPrice, (*tickerSlice)[bestIndex].count, time)
				*tickerSlice = remove(*tickerSlice, bestIndex)
			} else {
				(*tickerSlice)[bestIndex].count -= amt
				outputOrderExecuted((*tickerSlice)[bestIndex].id, activeID, (*tickerSlice)[bestIndex].exId, sellPrice, amt, time)
				amt = 0
			}
		}

		/*
			If amt = 0, order was settled
			If amt = prev, order found no match
			if amt < prev, order matched with resting and looks again
		*/
		return amt
	
        
    case 'S': 
		buyPrice := price
		bestIndex := -1
		amt := count

		// Find best match
		for i := len(*tickerSlice) - 1; i >= 0; i-- {
			if ((*tickerSlice)[i].cmd == 'B' && (*tickerSlice)[i].price >= buyPrice) {
				buyPrice = (*tickerSlice)[i].price
				bestIndex = i
			}
		}

		// If a match is found...
		if bestIndex != -1 {

			(*tickerSlice)[bestIndex].exId += 1

			// Active order has higher count
			if (amt >= (*tickerSlice)[bestIndex].count) {
				amt = amt - (*tickerSlice)[bestIndex].count
				outputOrderExecuted((*tickerSlice)[bestIndex].id, activeID, (*tickerSlice)[bestIndex].exId, buyPrice, (*tickerSlice)[bestIndex].count, time)
				*tickerSlice = remove(*tickerSlice, bestIndex)
			} else {
				(*tickerSlice)[bestIndex].count -= amt
				outputOrderExecuted((*tickerSlice)[bestIndex].id, activeID, (*tickerSlice)[bestIndex].exId, buyPrice, amt, time)
				amt = 0
			}
		}

		/*
			If amt = 0, order was settled
			If amt = prev, order found no match
			if amt < prev, order matched with resting and looks again
		*/
		return amt
	

    	default:
			fmt.Fprintf(os.Stderr, "Error in findMatch")
			return 0
	}
}

func handleOrder(order inputPackage, tickerSlice *[]CommandTuple) {
	in := order.in
	time := order.timestamp
	cmd := in.orderType
	id := in.orderId
	price := in.price
	num := in.count
	found := false
	if cmd == 'C' {
		for i := 0; i<len(*tickerSlice); i++ {
		    if ((*tickerSlice)[i].id == id) {
				outputOrderDeleted(in, true, time)
				found = true
				*tickerSlice = remove(*tickerSlice, i)
				break
			}
		}
		if !found {
			outputOrderDeleted(in, false, time)
		}
		return
	}
	for num > 0 {
		prevNum := num
		num = findMatch(cmd, price, num, id, tickerSlice, in.instrument, time)
		if (num == prevNum) {
			break
		}
	}

	if (num != 0) {
		tup := CommandTuple{cmd, id, price, num, 0}
		*tickerSlice = append(*tickerSlice, tup)
		outputOrderAdded(in, time)
	}

}

func readChannel(ch chan inputPackage) {
	tickerSlice := []CommandTuple{}
	for {
		select {
			case inputVar := <-ch:
			handleOrder(inputVar, &tickerSlice)
	    }
	}
}

func remove(slice []CommandTuple, index int) []CommandTuple {
    return append(slice[:index], slice[index+1:]...)
}
