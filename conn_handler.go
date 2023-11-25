package main

import (
	"fmt"
	"log"
	"net"
)

const (
	LOGIN_PROMPT = "\n##########\n" +
		"#\n" +
		"# Welcome to Chatroom Server\n" +
		"# Logged in as: User%d\n" +
		"#   ($exit to leave)\n" +
		"#\n" +
		"##########\n"
	CHATROOM_ENTER_PROMPT = "\n->\n" +
		"-> Welcome to Chat '%s'\n" +
		"->   ($exit to leave)\n" +
		"->\n> "
)

type ChatMessage struct {
	UserId  int
	Payload string
}

// A struct to handle the operations for each incoming Conn
// Conn is per-user
type ConnHandler struct {
	UserId               int
	ActiveChatId         int
	hasBeenStartedBefore bool // Because we handle the readFromConnLoop differently after starting it once
	conn                 net.Conn
	MsgCh                chan ChatMessage // Read incoming published messages from ChatroomManager
	PublishCh            chan ChatMessage // Publish messages to all subs of ChatroomManager
	UnsubCh              chan UnsubEvent  // To tell ChatroomManager to unsub us from chatroom
	QuitCh               chan struct{}
}

func NewConnHandler(userid int, conn net.Conn, unsubCh chan UnsubEvent, quitCh chan struct{}) *ConnHandler {
	return &ConnHandler{
		UserId:       userid,
		ActiveChatId: -1,
		conn:         conn,
		MsgCh:        make(chan ChatMessage),
		UnsubCh:      unsubCh,
		QuitCh:       quitCh,
	}
}

func (c *ConnHandler) startHandlingConn() {
	log.Printf("New connHandler for User%d\n", c.UserId)
	if !c.hasBeenStartedBefore {
		DebugPrint(fmt.Sprintf("%d NOT been started before", c.UserId))
		go c.readFromConnLoop()
		c.hasBeenStartedBefore = true
	} else {
		DebugPrint(fmt.Sprintf("%d HAS been started before", c.UserId))
	}
	var msg ChatMessage
	var formattedMsg string
	for {
		select {
		case msg = <-c.MsgCh:
			if c.UserId != msg.UserId {
				DebugPrint(fmt.Sprintf("  %d rcvd '%s'", c.UserId, msg.Payload))
				formattedMsg = fmt.Sprintf("\n[%d]: %s\n> ", msg.UserId, msg.Payload)
			} else {
				formattedMsg = "> "
			}
			c.conn.Write([]byte(formattedMsg))

		case <-c.QuitCh:
			log.Printf("closing connHandler%d\n", c.UserId)
			return
		}
	}
}

func (c *ConnHandler) readFromConnOnce() string {
	readCh := make(chan string)
	go func(readCh chan string) {
		buf := make([]byte, 2048)
		for {
			n, err := c.conn.Read(buf)
			if err != nil {
				log.Printf("%d Error during ReadOnce - %v\n", c.UserId, err)
				return
			}
			msg := string(buf[:n])
			if len(msg) <= 2 {
				c.conn.Write([]byte("\nInvalid input, please try again: "))
				continue
			} else {
				msg = msg[:len(msg)-2]
				readCh <- msg
				return
			}
		}
	}(readCh)

	select {
	case msg := <-readCh:
		return msg
	case <-c.QuitCh:
		return ""
	}
}

// Meant to be ran as a goroutine, we can stop/start readFromConnLoop this way
func (c *ConnHandler) resumeReadFromConnLoop() {
	c.readFromConnLoop()
}

// Read from Conn and broadcast to others through readCh
func (c *ConnHandler) readFromConnLoop() {
	log.Printf("readFromConn%d start\n", c.UserId)
	buf := make([]byte, 2048)
	stayActive := true
	for stayActive {
		n, err := c.conn.Read(buf)
		if err != nil {
			log.Println("Error during Read - ", err)
			continue
		}
		msg := string(buf[:n])

		// Don't spam new lines
		// TODO: This should be handled by the client, dont send messages unnecessarily
		if len(msg) > 2 {
			msg = msg[:len(msg)-2]
			if msg == "$exit" {
				c.PublishCh = nil // Stop reads from being published
				c.UnsubCh <- UnsubEvent{
					UserId: c.UserId,
					ChatId: c.ActiveChatId,
				}
				c.ActiveChatId = -1 // Set ActiveChatId to default as soon as we send the unsub event
				stayActive = false
			} else {
				if c.PublishCh != nil {
					c.PublishCh <- ChatMessage{
						UserId:  c.UserId,
						Payload: msg[:len(msg)-2],
					}
				}
			}
		}
	}
	log.Printf("readFromConnLoop(%d) stopped\n", c.UserId)
}
