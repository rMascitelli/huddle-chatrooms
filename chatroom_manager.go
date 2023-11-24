package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
)

type ChatroomManager struct {
	chatroom    ChatroomBroker
	subCh       chan *ConnHandler // Tx to chatroom about new subscriber
	unsubCh     chan int          // Rx from connHandler telling us to unsubscribe from chat
	newConnCh   chan net.Conn     // Rx from TCP Server telling us of new Conns
	activeConns map[int]*ConnHandler

	quitCh chan struct{}
	wg     *sync.WaitGroup

	// some DB-related stuff to save and load chats
}

func NewChatroomManager(quitCh chan struct{}, wg *sync.WaitGroup) ChatroomManager {
	cm := ChatroomManager{
		quitCh:      quitCh,
		subCh:       make(chan *ConnHandler),
		unsubCh:     make(chan int),
		activeConns: make(map[int]*ConnHandler),
		newConnCh:   make(chan net.Conn),
		wg:          wg,
	}
	cm.chatroom = *NewChatroomBroker(cm.quitCh, cm.subCh, cm.wg)
	return cm
}

func (cm *ChatroomManager) ListenForRequests() {
	log.Println("ChatroomMgr listening for Requests...")
	cm.wg.Add(1)
	for {
		select {
		case conn := <-cm.newConnCh:
			cm.HandleNewConnect(conn)
		case userId := <-cm.unsubCh:
			cm.chatroom.Unsubscribe(userId)
			cm.MoveExistingConnect(userId)
		case <-cm.quitCh:
			log.Println("ChatroomMgr shutting down...")
			cm.wg.Done()
			return
		}
	}
}

// TODO: Bug exists where server cannot stop while user is HandleNewConnect
// Could fix this by having ANOTHER (anonymous) channel to use here, launch a goroutine to listen to conn
//
//	If a selection is made from user, subscribe to channel
//	If quitCh is called, just return
//		(Make sure to cleanup goroutine)
func (cm *ChatroomManager) HandleNewConnect(conn net.Conn) {
	readErr := errors.New("")
	var msg string
	userId := rand.Intn(1000)
	ch := NewConnHandler(userId, conn, cm.wg, cm.unsubCh, cm.quitCh)
	conn.Write([]byte(fmt.Sprintf(LOGIN_PROMPT, userId)))
	conn.Write([]byte("\nWhich chatroom would you like? "))
	for readErr != nil {
		msg, readErr = ch.readFromConnOnce()
		if readErr != nil {
			log.Println("Err reading:", readErr)
		}
	}
	log.Println("Changing to chat:", msg)
	ch.conn.Write([]byte("\n> "))
	cm.activeConns[userId] = ch
	cm.chatroom.subCh <- ch
}

func (cm *ChatroomManager) MoveExistingConnect(userId int) {
	readErr := errors.New("")
	var msg string
	ch := cm.activeConns[userId]
	ch.conn.Write([]byte("Which chatroom to change to? "))
	for readErr != nil {
		msg, readErr = ch.readFromConnOnce()
		if readErr != nil {
			log.Println("Err reading:", readErr)
			ch.conn.Write([]byte("\nInvalid input, please try again: "))
		}
	}
	log.Println("Changing to chat:", msg)
	go ch.resumeReadFromConnLoop()
	ch.conn.Write([]byte("\n> "))

	// TODO: Change cm.chatroom to be array of chatrooms
	//		Allow user to choose chatroom
	cm.chatroom.subCh <- ch
}
