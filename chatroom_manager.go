package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
)

type ChatroomManager struct {
	chatroom    ChatroomBroker
	subCh       chan ConnHandler // Tx to chatroom about new subscriber
	unsubCh     chan int         // Rx from connHandler telling us to unsubscribe from chat
	newConnCh   chan net.Conn    // Rx from TCP Server telling us of new Conns
	activeConns map[int]ConnHandler

	quitCh chan struct{}
	wg     *sync.WaitGroup

	// some DB-related stuff to save and load chats
}

func NewChatroomManager(quitCh chan struct{}, wg *sync.WaitGroup) ChatroomManager {
	cm := ChatroomManager{
		quitCh:      quitCh,
		subCh:       make(chan ConnHandler),
		unsubCh:     make(chan int),
		activeConns: make(map[int]ConnHandler),
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
		log.Println("await req...")
		select {
		case conn := <-cm.newConnCh:
			cm.HandleNewConnect(conn)
		case userId := <-cm.unsubCh:
			cm.chatroom.Unsubscribe(userId)
			cm.MoveExistingConnect(userId)
		case <-cm.quitCh:
			cm.wg.Done()
		}
	}
}

func (cm *ChatroomManager) HandleNewConnect(conn net.Conn) {
	userId := rand.Intn(1000)
	conn.Write([]byte(fmt.Sprintf(LOGIN_PROMPT, userId)))
	conn.Write([]byte("Which chatroom would you like?\n> "))
	ch := *NewConnHandler(userId, conn, cm.wg, cm.unsubCh, cm.quitCh)
	cm.activeConns[userId] = ch
	cm.chatroom.subCh <- ch
}

func (cm *ChatroomManager) MoveExistingConnect(userId int) {
	ch := cm.activeConns[userId]
	ch.conn.Write([]byte("Which chatroom would you like?\n> "))

	// TODO: Change cm.chatroom to be array of chatrooms
	//		Allow user to choose chatroom
	//cm.chatroom.subCh <- ch
}
