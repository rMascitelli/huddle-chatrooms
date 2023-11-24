package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const (
	DEBUG_MODE = true
)

func DebugPrint(s string) {
	if DEBUG_MODE {
		fmt.Println(s)
	}
}

type TCPServer struct {
	listenAddr  string
	ln          net.Listener
	chatroomMgr ChatroomManager
	sigCh       chan os.Signal // For main server to accept SIGTERM
	quitCh      chan struct{}
	subCh       chan net.Conn // Accept ConnHandler to add to subs
	unsubCh     chan int      // Accept ConnHandler to del from subs
	wg          *sync.WaitGroup
}

func NewTCPServer(listenAddr string) *TCPServer {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	wg := sync.WaitGroup{}
	tcp := &TCPServer{
		listenAddr: listenAddr,
		sigCh:      sigCh,
		quitCh:     make(chan struct{}),
		subCh:      make(chan net.Conn),
		unsubCh:    make(chan int),
		wg:         &wg,
	}
	tcp.chatroomMgr = NewChatroomManager(tcp.quitCh, tcp.wg)
	return tcp
}

func (t *TCPServer) Start() {
	ln, err := net.Listen("tcp", t.listenAddr)
	if err != nil {
		log.Println("Failed to create listener for server")
	}
	t.ln = ln
	defer ln.Close()

	// Start accepting and serving connections
	go t.chatroomMgr.ListenForRequests()
	go t.chatroomMgr.chatroom.Start() // TODO: This would be a loop to start all chatrooms
	go t.acceptLoop()
	log.Println("Server listening at", t.listenAddr)

	// Wait for cancel signal, close listener, and wait for goroutines to finish
	<-t.sigCh
	close(t.quitCh)
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
		log.Println("New conn from", conn.RemoteAddr())

		t.chatroomMgr.newConnCh <- conn
	}
}

func main() {
	t := NewTCPServer(":8080")
	t.Start()
}
