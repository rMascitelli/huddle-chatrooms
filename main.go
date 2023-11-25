package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const (
	DEBUG_MODE = 0
)

func DebugPrint(s string) {
	if DEBUG_MODE > 0 {
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
}

func NewTCPServer(listenAddr string) *TCPServer {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	tcp := &TCPServer{
		listenAddr: listenAddr,
		sigCh:      sigCh,
		quitCh:     make(chan struct{}),
		subCh:      make(chan net.Conn),
		unsubCh:    make(chan int),
	}
	tcp.chatroomMgr = NewChatroomManager(tcp.quitCh)
	return tcp
}

func (t *TCPServer) Start() {
	ln, err := net.Listen("tcp", t.listenAddr)
	if err != nil {
		log.Println("Failed to create listener for server")
	}
	t.ln = ln
	defer ln.Close()

	// Start all chatrooms and chatroomMgr
	for _, chatroom := range t.chatroomMgr.chatroomIndex {
		go chatroom.Start()
	}
	go t.chatroomMgr.listenForRequests()

	// Start accepting and serving connections
	go t.acceptLoop()
	log.Println("Server listening at", t.listenAddr)

	// Wait for cancel signal, close listener
	<-t.sigCh
	close(t.quitCh)
	log.Println("Shutting down TCP server!")
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
