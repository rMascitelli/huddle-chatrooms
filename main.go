package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
)

type TCPServer struct {
	listenAddr string
	ln         net.Listener
	cancelCh   chan os.Signal
}

func NewTCPServer(listenAddr string) *TCPServer {
	cancelCh := make(chan os.Signal, 1)
	signal.Notify(cancelCh, syscall.SIGTERM, syscall.SIGINT)
	return &TCPServer{
		listenAddr: listenAddr,
		cancelCh:   cancelCh,
	}
}

func (t *TCPServer) Start() {
	ln, err := net.Listen("tcp", t.listenAddr)
	if err != nil {
		log.Println("Failed to create listener for server")
	}
	defer ln.Close()
	t.ln = ln

	go t.acceptLoop()

	log.Println("Server listening at", t.listenAddr)
	sig := <-t.cancelCh
	log.Printf("Caught signal %v", sig)
}

func (t *TCPServer) acceptLoop() {
	for {
		conn, err := t.ln.Accept()
		if err != nil {
			log.Println("Error during Accept - ", err)
			continue
		}
		log.Println("Accepted new conn - ", conn.RemoteAddr())

		go t.readLoop(conn)
	}
}

func (t *TCPServer) readLoop(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Println("Error during Read - ", err)
			continue
		}

		msg := string(buf[:n])
		log.Println(msg)
		_, err = conn.Write([]byte(msg + "\n"))
	}
}

func main() {
	t := NewTCPServer(":8080")
	t.Start()
}
