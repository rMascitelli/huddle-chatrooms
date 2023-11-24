package main

import (
	"fmt"
	"log"
	"net"
	"sync"
)

const (
	LOGIN_PROMPT = "\n##########\n" +
		"#\n" +
		"# Welcome to HuddleChats\n" +
		"#\n" +
		"##########\n\n> "
	PROMPT = "> "
)

type ChatMessage struct {
	UserId  int
	Payload string
}

// A struct to handle the operations for each incoming Conn
// Conn is per-user
type ConnHandler struct {
	Userid    int
	conn      net.Conn
	wg        *sync.WaitGroup
	MsgCh     chan ChatMessage // Read incoming published messages from ChatroomManager
	PublishCh chan ChatMessage // Publish messages to all subs of ChatroomManager
	QuitCh    chan struct{}
}

func NewConnHandler(userid int, conn net.Conn, wg *sync.WaitGroup, publishCh chan ChatMessage, quitCh chan struct{}) *ConnHandler {
	return &ConnHandler{
		Userid:    userid,
		conn:      conn,
		wg:        wg,
		MsgCh:     make(chan ChatMessage),
		PublishCh: publishCh,
		QuitCh:    quitCh,
	}
}

func (c *ConnHandler) startHandlingConn() {
	log.Printf("New connHandler for User%d\n", c.Userid)
	go c.readFromConnLoop(c.conn)
	c.wg.Add(1)
	var msg ChatMessage

	for {
		select {
		case msg = <-c.MsgCh:
			DebugPrint(fmt.Sprintf("  %d rcvd '%s'", c.Userid, msg.Payload))
			formatted_msg := fmt.Sprintf("[%d]: %s\n", msg.UserId, msg.Payload)
			c.conn.Write([]byte(formatted_msg))

		case <-c.QuitCh:
			log.Println("closing connHandler")
			c.wg.Done()
			return
		}
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
		msg := string(buf[:n])

		// Don't spam new lines
		if len(msg) > 2 {
			c.PublishCh <- ChatMessage{
				UserId:  c.Userid,
				Payload: msg[:len(msg)-2],
			}
		}
	}
}
