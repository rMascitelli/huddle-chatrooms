package main

import (
	"log"
	"net"
	"sync"
)

// A struct to handle the operations for each incoming Conn
// Conn is per-user
type ConnHandler struct {
	// Unique per handler
	userid int
	conn   net.Conn

	// Common among all
	readCh chan []byte
	quitCh chan bool
	wg     *sync.WaitGroup
}

func NewConnHandler(userid int, conn net.Conn, readCh chan []byte, quitCh chan bool, wg *sync.WaitGroup) *ConnHandler {
	return &ConnHandler{
		userid: userid,
		conn:   conn,
		readCh: readCh,
		quitCh: quitCh,
		wg:     wg,
	}
}

func (c *ConnHandler) startHandlingConn() {
	log.Println("starting connHandler")
	c.wg.Add(1)
	go c.readFromChLoop(c.conn)
	go c.readFromConnLoop(c.conn)

	// Wait for a signal from quitCh, then close conn + signal wg Done
	<-c.quitCh
	log.Println("closing connHandler")
	c.wg.Done()
}

func (c *ConnHandler) readFromChLoop(conn net.Conn) {
	for msg := range c.readCh {
		log.Println("Got msg - ", string(msg))
		// TODO: If msg.userid != c.userid, send msg over conn
	}
}

// Read from Conn and broadcast to others through readCh
func (c *ConnHandler) readFromConnLoop(conn net.Conn) {
	defer c.conn.Close()
	buf := make([]byte, 2048)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			log.Println("Error during Read - ", err)
			continue
		}

		c.readCh <- buf[:n]
	}
}
