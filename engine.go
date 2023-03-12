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

func (e *Engine) accept(ctx context.Context, conn net.Conn, writeCh chan <- string, newClientCh chan <- chan InstrumentChannel) {
	go func() {
		<-ctx.Done()
		conn.Close()
	}()
	go handleConn(conn, writeCh, newClientCh)
}

func handleConn(conn net.Conn, writeCh chan <- string, newClientCh chan <- chan InstrumentChannel) {
	defer conn.Close()
	instrumentChMap := make(map[string] chan input)
	idMap := make(map[uint32] string)
	readCh := make(chan InstrumentChannel)
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
			fmt.Fprintf(os.Stderr, "before switch")
			switch in.orderType {
			case inputCancel:
				fmt.Fprintf(os.Stderr, "Got cancel ID: %v\n", in.orderId)
				//outputOrderDeleted(in, true, GetCurrentTimestamp())
				instrument := idMap[in.orderId]
				write := instrumentChMap[instrument]
				write <- in
			default:
				idMap[in.orderId] = in.instrument
				fmt.Fprintf(os.Stderr, "default")
				fmt.Fprintf(os.Stderr, "Got order: %c %v x %v @ %v ID: %v\n",
					in.orderType, in.instrument, in.count, in.price, in.orderId)
				write, ok := instrumentChMap[in.instrument] 
				if (!ok) {
					writeCh <- in.instrument //1
					newCh := <- readCh
					instrumentChMap[newCh.instrumentName] = newCh.channel
					write = newCh.channel
				}
				write <- in //4

			}
			
			}
	}
}

func GetCurrentTimestamp() int64 {
	return time.Now().UnixNano()
}

func findMatch(cmd inputType, price uint32, count uint32, activeID uint32, tickerSlice *[]CommandTuple) uint32 {
	fmt.Fprintf(os.Stderr, "findMatch\n")
	fmt.Fprintf(os.Stderr, "%s", cmd)

    switch cmd {
		
    case 'B': 
		fmt.Fprintf(os.Stderr, "reached buy case")
		sellPrice := price
		bestIndex := -1
		amt := count

		// Find best match

		for i := 0; i < len(*tickerSlice); i++ {
			if ((*tickerSlice)[i].cmd == 'S' && (*tickerSlice)[i].price <= price) {
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
				outputOrderExecuted((*tickerSlice)[bestIndex].id, activeID, (*tickerSlice)[bestIndex].exId, sellPrice, (*tickerSlice)[bestIndex].count, GetCurrentTimestamp())
				*tickerSlice = remove(*tickerSlice, bestIndex)
			} else {
				(*tickerSlice)[bestIndex].count -= amt
				outputOrderExecuted((*tickerSlice)[bestIndex].id, activeID, (*tickerSlice)[bestIndex].exId, sellPrice, amt, GetCurrentTimestamp())
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
		fmt.Fprintf(os.Stderr, "Length before for loop: %d", len(*tickerSlice))
		for i := 0; i < len(*tickerSlice); i++ {
			if ((*tickerSlice)[i].cmd == 'B' && (*tickerSlice)[i].price >= price) {
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
				outputOrderExecuted((*tickerSlice)[bestIndex].id, activeID, (*tickerSlice)[bestIndex].exId, buyPrice, (*tickerSlice)[bestIndex].count, GetCurrentTimestamp())
				*tickerSlice = remove(*tickerSlice, bestIndex)
			} else {
				(*tickerSlice)[bestIndex].count -= amt
				outputOrderExecuted((*tickerSlice)[bestIndex].id, activeID, (*tickerSlice)[bestIndex].exId, buyPrice, amt, GetCurrentTimestamp())
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
        fmt.Println("Invalid command type")
		return 0
    }
}

func handleOrder(in input, tickerSlice *[]CommandTuple) {
	fmt.Fprintf(os.Stderr, "handleOrder")
	cmd := in.orderType
	id := in.orderId
	price := in.price
	num := in.count
	for num > 0 {
		prevNum := num
		num = findMatch(cmd, price, num, id, tickerSlice)
		if (num == prevNum) {
			break
		}
	}

	if (num != 0) {
		tup := CommandTuple{cmd, id, price, num, 0}
		fmt.Fprintf(os.Stderr, "Before appending length: %s", len(*tickerSlice))
		*tickerSlice = append(*tickerSlice, tup)
		fmt.Fprintf(os.Stderr, "After Appending Length: %s", len(*tickerSlice))
		outputOrderAdded(in, GetCurrentTimestamp())
	}

}

func readChannel(ch chan input) {
	fmt.Fprintf(os.Stderr, "readChannel")
	tickerSlice := []CommandTuple{}
	for {
		select {
			case inputVar := <-ch:
			fmt.Fprintf(os.Stderr, "\nActually Read from channel\n")
			handleOrder(inputVar, &tickerSlice)
	  }
	}
}

func remove(slice []CommandTuple, index int) []CommandTuple {
    return append(slice[:index], slice[index+1:]...)
}
