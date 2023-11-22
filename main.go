package main

import (
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type TCPServer struct {
	listenAddr string
	ln         net.Listener
	sigCh      chan os.Signal // For main server to accept SIGTERM
	readCh     chan Message   // For all conns to read/write to eachother
	quitCh     chan bool      // Channel specifically for waiting for sigCh signal
	wg         sync.WaitGroup // WaitGroup to wait for all goroutines to finish
}

func NewTCPServer(listenAddr string) *TCPServer {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	return &TCPServer{
		listenAddr: listenAddr,
		sigCh:      sigCh,
		readCh:     make(chan Message, 10),
		quitCh:     make(chan bool),
		wg:         sync.WaitGroup{},
	}
}

func (t *TCPServer) Start() {
	ln, err := net.Listen("tcp", t.listenAddr)
	if err != nil {
		log.Println("Failed to create listener for server")
	}
	t.ln = ln

	// Start accepting and serving connections
	go t.acceptLoop()
	log.Println("Server listening at", t.listenAddr)

	// Wait for cancel signal, close listener, and wait for goroutines to finish
	<-t.sigCh
	close(t.quitCh)
	ln.Close()
	t.wg.Wait()
	log.Println("all goroutines complete!")
}

func (t *TCPServer) acceptLoop() {
	for {
		conn, err := t.ln.Accept()
		if err != nil {
			select {
			case <-t.quitCh:
				return
			default:
				log.Println("Error during Accept - ", err)
				continue
			}
		}
		log.Println("Accepted new conn - ", conn.RemoteAddr())

		NewConnHandler(rand.Intn(1000), conn, t.readCh, t.quitCh, &t.wg).startHandlingConn()
	}
}

func main() {
	t := NewTCPServer(":8080")
	t.Start()
}
