package main

import (
	"log"
	"net"
	"sync"
)

type Message struct {
	Userid  int
	Payload string
}

// A struct to handle the operations for each incoming Conn
// Conn is per-user
type ConnHandler struct {
	// Unique per handler
	userid int
	conn   net.Conn

	// Common among all
	readCh chan Message
	quitCh chan bool
	wg     *sync.WaitGroup
}

func NewConnHandler(userid int, conn net.Conn, readCh chan Message, quitCh chan bool, wg *sync.WaitGroup) *ConnHandler {
	return &ConnHandler{
		userid: userid,
		conn:   conn,
		readCh: readCh,
		quitCh: quitCh,
		wg:     wg,
	}
}

func (c *ConnHandler) startHandlingConn() {
	log.Printf("starting new connHandler for User%d\n", c.userid)
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
		log.Printf("[%d]: %s", msg.Userid, msg.Payload)
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
		c.readCh <- Message{
			Userid:  c.userid,
			Payload: string(buf[:n]),
		}
	}
}
