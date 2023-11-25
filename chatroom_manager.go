package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
)

type ChatroomManager struct {
	chatroomIndex map[int]*ChatroomBroker
	subCh         chan *ConnHandler // Tx to chatroom about new subscriber
	unsubCh       chan UnsubEvent   // Rx from connHandler telling us to unsubscribe from chat
	newConnCh     chan net.Conn     // Rx from TCP Server telling us of new Conns
	activeConns   map[int]*ConnHandler

	quitCh chan struct{}

	// some DB-related stuff to save and load chats
}

type UnsubEvent struct {
	UserId int
	ChatId int
}

func NewChatroomManager(quitCh chan struct{}) ChatroomManager {
	cm := ChatroomManager{
		quitCh:        quitCh,
		subCh:         make(chan *ConnHandler),
		unsubCh:       make(chan UnsubEvent),
		activeConns:   make(map[int]*ConnHandler),
		newConnCh:     make(chan net.Conn),
		chatroomIndex: make(map[int]*ChatroomBroker),
	}
	cm.createDefaultChatrooms()
	return cm
}

func (cm *ChatroomManager) createDefaultChatrooms() {
	cm.chatroomIndex[1] = NewChatroomBroker("Test1", 1, cm.quitCh)
	cm.chatroomIndex[2] = NewChatroomBroker("Test2", 2, cm.quitCh)
	cm.chatroomIndex[3] = NewChatroomBroker("Test3", 3, cm.quitCh)
	cm.chatroomIndex[4] = NewChatroomBroker("SpookyMonsterChat", 4, cm.quitCh)
}

func (cm *ChatroomManager) listenForRequests() {
	log.Println("ChatroomMgr listening for Requests...")
	for {
		select {
		case conn := <-cm.newConnCh:
			cm.HandleNewConnectPrompt(conn)
		case unsubEvent := <-cm.unsubCh:
			log.Printf("User%d left chat '%s'\n", unsubEvent.UserId, cm.chatroomIndex[unsubEvent.ChatId].ChatroomName)
			delete(cm.chatroomIndex[unsubEvent.ChatId].subs, unsubEvent.UserId)
			cm.MoveExistingConnectPrompt(unsubEvent.UserId)
		case <-cm.quitCh:
			log.Println("ChatroomMgr shutting down...")
			return
		}
	}
}

func (cm *ChatroomManager) startNewChatroomLoop() {
	for {
		select {
		case <-cm.quitCh:
			return
		}
	}
}

func (cm *ChatroomManager) HandleNewConnectPrompt(conn net.Conn) {
	userId := rand.Intn(1000)
	ch := NewConnHandler(userId, conn, cm.unsubCh, cm.quitCh)
	conn.Write([]byte(fmt.Sprintf(LOGIN_PROMPT, userId)))
	conn.Write([]byte(cm.ReadAllChatrooms()))
	conn.Write([]byte("Which chatroom would you like? "))
	var msg string
	for {
		msg = ch.readFromConnOnce()
		if msg != "" {
			if msg == "$exit" {
				ch.conn.Write([]byte("\nThanks for coming!\n"))
				delete(cm.activeConns, userId)
				ch.conn.Close()
				return
			}
			chosenChatId, err := strconv.Atoi(msg)
			if err == nil && cm.DoesChatroomExist(chosenChatId) {
				chatroomName := cm.chatroomIndex[chosenChatId].ChatroomName
				log.Printf("User%d changing to chat: %s\n", ch.UserId, chatroomName)
				ch.conn.Write([]byte(fmt.Sprintf("Changing to chat '%s'...\n", chatroomName)))
				ch.conn.Write([]byte(fmt.Sprintf(CHATROOM_ENTER_PROMPT, chatroomName)))
				cm.activeConns[userId] = ch
				cm.chatroomIndex[chosenChatId].subCh <- ch
				return
			}
			ch.conn.Write([]byte(fmt.Sprintf("\nChat '%s' not found, please try again: ", msg)))
		} else {
			return // Received SIGTERM
		}
	}
}

func (cm *ChatroomManager) MoveExistingConnectPrompt(userId int) {
	ch := cm.activeConns[userId]
	ch.conn.Write([]byte(cm.ReadAllChatrooms()))
	ch.conn.Write([]byte("Which chatroom to change to? "))
	var msg string
	for {
		msg = ch.readFromConnOnce()
		if msg != "" {
			if msg == "$exit" {
				ch.conn.Write([]byte("\nThanks for coming!\n"))
				delete(cm.activeConns, userId)
				ch.conn.Close()
				return
			}
			chosenChatId, err := strconv.Atoi(msg) // Remove newline
			if err == nil && cm.DoesChatroomExist(chosenChatId) {
				chatroomName := cm.chatroomIndex[chosenChatId].ChatroomName
				log.Printf("User%d changing to chat: %s\n", ch.UserId, chatroomName)
				ch.conn.Write([]byte(fmt.Sprintf("Changing to chat '%s'...\n", chatroomName)))
				ch.conn.Write([]byte(fmt.Sprintf(CHATROOM_ENTER_PROMPT, chatroomName)))
				go ch.resumeReadFromConnLoop()
				cm.chatroomIndex[chosenChatId].subCh <- ch
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
	cm.chatroomIndex[chatId] = NewChatroomBroker(chatroomName, chatId, cm.quitCh)
	// TODO: How can I launch a new chatroom? cant just goroutine here because function will exit
	//		Maybe goroutine waiting on NewChatCh...
	//			When it receives *ChatroomBroker,
	return nil
}

func (cm *ChatroomManager) ReadAllChatrooms() string {
	reportString := "\n# Available chats:\n[\n"
	for _, chatroom := range cm.chatroomIndex {
		reportString += fmt.Sprintf("  -> [%d] %s\n", chatroom.ChatId, chatroom.ChatroomName)
	}
	reportString += "  -> [x] Create new chatroom\n"
	reportString += "]\n"
	return reportString
}

func (cm *ChatroomManager) DoesChatroomExist(chatId int) bool {
	_, ok := cm.chatroomIndex[chatId]
	return ok
}
