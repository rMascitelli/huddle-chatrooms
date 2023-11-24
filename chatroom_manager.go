package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
)

type ChatroomManager struct {
	chatroom      ChatroomBroker
	chatroomIndex map[int]*ChatroomBroker
	subCh         chan *ConnHandler // Tx to chatroom about new subscriber
	unsubCh       chan int          // Rx from connHandler telling us to unsubscribe from chat
	newConnCh     chan net.Conn     // Rx from TCP Server telling us of new Conns
	activeConns   map[int]*ConnHandler

	quitCh chan struct{}

	// some DB-related stuff to save and load chats
}

type UnsubEvent struct {
	userId   int
	curChat  int
	destChat int
}

func NewChatroomManager(quitCh chan struct{}) ChatroomManager {
	cm := ChatroomManager{
		quitCh:        quitCh,
		subCh:         make(chan *ConnHandler),
		unsubCh:       make(chan int),
		activeConns:   make(map[int]*ConnHandler),
		newConnCh:     make(chan net.Conn),
		chatroomIndex: make(map[int]*ChatroomBroker),
	}
	cm.createDefaultChatrooms()
	return cm
}

func (cm *ChatroomManager) createDefaultChatrooms() {
	cm.chatroomIndex[1] = NewChatroomBroker("Test1", 1, cm.quitCh, cm.subCh)
	cm.chatroomIndex[2] = NewChatroomBroker("Test2", 2, cm.quitCh, cm.subCh)
	cm.chatroomIndex[3] = NewChatroomBroker("Test3", 3, cm.quitCh, cm.subCh)
	cm.chatroomIndex[4] = NewChatroomBroker("SpookyMonsterChat", 4, cm.quitCh, cm.subCh)
}

func (cm *ChatroomManager) ListenForRequests() {
	log.Println("ChatroomMgr listening for Requests...")
	for {
		select {
		case conn := <-cm.newConnCh:
			cm.HandleNewConnect(conn)
		case userId := <-cm.unsubCh:
			cm.chatroom.Unsubscribe(userId)
			cm.MoveExistingConnect(userId)
		case <-cm.quitCh:
			log.Println("ChatroomMgr shutting down...")
			return
		}
	}
}

func (cm *ChatroomManager) HandleNewConnect(conn net.Conn) {
	userId := rand.Intn(1000)
	ch := NewConnHandler(userId, conn, cm.unsubCh, cm.quitCh)
	conn.Write([]byte(fmt.Sprintf(LOGIN_PROMPT, userId)))
	conn.Write([]byte(cm.ReadAllChatrooms()))
	conn.Write([]byte("Which chatroom would you like? "))
	var msg string
	for {
		msg = ch.readFromConnOnce()
		if msg != "" {
			msg = msg[:len(msg)-2] // Remove newline
			if doesExist, chatId := cm.DoesChatroomExist(msg); doesExist {
				chatroomName := cm.chatroomIndex[chatId].ChatroomName
				log.Println("Changing to chat:", chatroomName)
				ch.conn.Write([]byte(fmt.Sprintf("Changing to chat '%s'...\n", chatroomName)))
				ch.conn.Write([]byte(fmt.Sprintf(CHATROOM_ENTER_PROMPT, chatroomName)))
				cm.activeConns[userId] = ch
				cm.chatroomIndex[chatId].subCh <- ch
				return
			}
			ch.conn.Write([]byte(fmt.Sprintf("\nChat '%s' not found, please try again: ", msg)))
		} else {
			return // Received SIGTERM
		}
	}
}

func (cm *ChatroomManager) MoveExistingConnect(userId int) {
	ch := cm.activeConns[userId]
	ch.conn.Write([]byte(cm.ReadAllChatrooms()))
	ch.conn.Write([]byte("Which chatroom to change to? "))
	var msg string
	for {
		msg = ch.readFromConnOnce()
		if msg != "" {
			msg = msg[:len(msg)-2] // Remove newline
			if doesExist, chatId := cm.DoesChatroomExist(msg); doesExist {
				chatroomName := cm.chatroomIndex[chatId].ChatroomName
				log.Println("Changing to chat:", chatroomName)
				ch.conn.Write([]byte(fmt.Sprintf("Changing to chat '%s'...\n", chatroomName)))
				ch.conn.Write([]byte(fmt.Sprintf(CHATROOM_ENTER_PROMPT, chatroomName)))
				go ch.resumeReadFromConnLoop()
				log.Println("WRITING TO SUBCH-", chatroomName)
				cm.chatroomIndex[chatId].subCh <- ch
				log.Println("DONE HANDLING EXISTING CONN")
				return
			}
			ch.conn.Write([]byte(fmt.Sprintf("\nChat '%s' not found, please try again: ", msg)))
		} else {
			return // Received SIGTERM
		}
	}
}

func (cm *ChatroomManager) AddNewChatroom(chatroomName string, chatId int) error {
	if _, ok := cm.chatroomIndex[chatId]; ok {
		return errors.New("chatroom already exists")
	}
	cm.chatroomIndex[chatId] = NewChatroomBroker(chatroomName, chatId, cm.quitCh, cm.subCh)
	// TODO: How can I launch a new chatroom? cant just goroutine here because function will exit
	//		Maybe goroutine waiting on NewChatCh...
	//			When it receives *ChatroomBroker,
	return nil
}

func (cm *ChatroomManager) ReadAllChatrooms() string {
	reportString := "\n# Available chats:\n[\n"
	for _, chatroom := range cm.chatroomIndex {
		reportString += fmt.Sprintf("  -> %s\n", chatroom.ChatroomName)
	}
	reportString += "]\n"
	return reportString
}

func (cm *ChatroomManager) DoesChatroomExist(chatName string) (bool, int) {
	for chatId, chatroom := range cm.chatroomIndex {
		if chatroom.ChatroomName == chatName {
			return true, chatId
		}
	}
	return false, -1
}
