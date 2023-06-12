# Exchange Matching Engine

This Go program uses techniques of concurrent programming to efficiently match buy and sell orders for a stock exchange. As active orders are placed, they will 
be compared to all resting orders to find a potential match. If there is no match, the active order is placed into a resting order book and can be matched at any point
in the future. The engine matches orders using the price-time priority rule. This rule for matching two orders on an exchange is expressed using the following conditions – which must all be true for the matching to happen:

- The side of the two orders must be different (i.e. a buy order must match against a sell orders or vice versa).
- The instrument of the two orders must be the same (i.e. an order for “GOOG” must match against another order for “GOOG”).
- The size of the two orders must be greater than zero.
- The price of the buy order must be greater or equal to the price of the sell order.
- In case multiple orders can be matched, the order with the best price is matched first. For sell orders, the best price is the lowest; for buy orders, the best price is the highest.
- If there are still multiple matchable orders, the order that was added to the order book the earliest (ie. the order whose “added to order book” log has the earliest timestamp) will be matched first. 

# How to Use

Clone the repository and run "make", which will compile the latest code into an engine executable and client executable. You can start the engine using `./engine <path_to_socket>`.
This will start a server and listen on the socket at `<path_to_socket>`. To start a client, run `./client <path_to_socket>`. You can start as many clients as you'd like, and each can send
orders in parallel that will be processed concurrently.

## Inputs

There are three input commands that can be sent by the client:

#### New Buy Order
Arguments: Order ID, Instrument, Price, Count
Example: `B 123 GOOG 2700 10`

#### New Sell Order
Arguments: Order ID, Instrument, Price, Count
Example: `S 124 GOOG 1800 8`

#### Cancel Order
Arguments: Order ID
Note: Cancels may only be sent by the same client that placed the order.
Example: `C 123`

## Outputs

These outputs will be sent to stdout by the engine executable.

#### Order Successfully Added
Format: `<B/S> <Order ID> <Instrument> <Price> <Count> <Timestamp completed>`
Example: `B 123 GOOG 2700 10 1`
Example: `S 124 GOOG 1800 8 2`

#### Order Executed
Format: `E <Resting order ID> <New order ID> <Execution ID> <Price> <Count> <Timestamp completed>`
Example: `E 123 124 1 2700 8 3`

#### Order Deleted
Format: `X <Order ID> <A/R> <Timestamp completed>`
A represents an accepted cancel, and R represents a rejected cancel.
Example: `X 123 A 4`

# Explanation of Concurrency

We use one goroutine for each client and one goroutine for each instrument to enable concurrency. We also have a main goroutine which coordinates with the clients to spawn instrument goroutines as needed and maintain synchronization of channels between client goroutines and instrument goroutines. The main goroutine is run as an anonymous function in main.go, and it maintains a master source of information that all client goroutines draw from. It owns two channels that it reads from, clientReadCh and newClientCh. 

Initializing Client Goroutines - newClientCh is a buffered higher order channel that any client can write to, and is passed to a new client through the accept method (2). Whenever that new client is initiated, it makes a new InstrumentChannel struct (used to send information about instrument channels) and sends the channel via newClientCh to the main goroutine (3). The main goroutine stores that channel in a slice of client channels, so that whenever a new instrument is initiated, it can send information about that instrument to every client (4). Furthermore, it sends information about all current instruments in the master hashmap over that channel to the newly initiated client, so that the client has a fully updated set of data (5). 

Initializing Instrument Goroutines - Whenever a client receives a new order (6), it first checks if it already has that instrument stored in its hashmap. If it does, then it will process that order by searching for a matching order in its slice. If not, it will send the new instrument name as a string to the main goroutine via the clientReadCh (8). When the main goroutine receives a new instrument name on the clientReadCh and that instrument is not currently in its hashmap, it will spawn a new goroutine for the instrument (9). It will also create a channel for that instrument of type inputPackage, which stores all relevant information pertaining to an incoming input (including its timestamp). It then broadcasts a message of type InstrumentChannel struct to every client channel in its ClientWriteChSlice (10). Every client goroutine then receives that message and updates its instrument hashmap with the instrument instrument as a key and the instrument channel as the value (11).

Handling Orders - When a client receives an order and does not have the instrument in its hashmap, it runs the above instrument goroutine initiation process, and blocks until the instrument is added to the hashmap. Then, it will process the order by retrieving the proper instrument channel from its hashmap and sending an input package (consisting of the input and current timestamp) to the instrument goroutine (13). The instrument goroutine takes in orders one at a time, and processes them by matching against a slice of resting orders. If a match is found, the corresponding output is printed and the relevant counts are updated (13). Cancels are sent by the client in the same manner, except that the client will first look up the cancel id in a hashmap that maps ids to instrument strings, and then sends the cancel id to the corresponding instrument channel.

This implementation achieves Instrument-Level Concurrency. Each client can receive input independently, and has its own hashmap of instrument channels. These channels are kept consistent across all clients via broadcasts from the main channel, which stores a master source of information about the instruments. Different instruments can run concurrently because each instrument has its own goroutine that orders are sent to for that instrument only. However, orders for the same instrument are serialized because each instrument only takes in one order at a time from its input channel. 

## Testing

We began by testing basic functionality against simple one-thread test cases, which highlighted several pointer and logical errors that were patched. We passed the basic cases and then moved on to manual testing with multiple threads. In an environment with 4 threads, our engine was able to perform cross-thread full and partial matching. The engine also maintained correct ordering of matching based on pricing and timestamp. The engine also only allowed cancellations for orders produced within the same thread.

We then moved on to creating complex test cases using the Python script generate_test_cases.py. We generated test files to mimic complex testing cases. We also created two more categories of tests: medium (up to 4 clients) and mediumHard (up to 20 clients). Below are parameters for our complex test cases.

- Random stock instrument chosen from a group of length 428
- 40 clients (and therefore 40 concurrent threads)
- Random number of orders in range [1000, 50000]
- Random order type, both buy, sell, and cancel with a probability of ⅓ each
- Random assignment of client with an equal probability for all clients
- Random price in range [100, 2000]
- Random count between [10, 1000]

