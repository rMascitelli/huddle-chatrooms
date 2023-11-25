package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"time"
)

type ChatroomManager struct {
	chatroomIndex map[int]*ChatroomBroker
	activeConns   map[int]*ConnHandler
	rwLock        sync.Mutex
	subCh         chan *ConnHandler // Tx to chatroom about new subscriber
	unsubCh       chan UnsubEvent   // Rx from connHandler telling us to unsubscribe from chat
	newConnCh     chan net.Conn     // Rx from TCP Server telling us of new Conns
	createChatCh  chan string
	quitCh        chan struct{}

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
		createChatCh:  make(chan string),
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

func (cm *ChatroomManager) Start() {
	log.Println("Starting ChatroomManager...")
	go cm.startListenForRequests()
	<-cm.quitCh
}

func (cm *ChatroomManager) startListenForRequests() {
	log.Println("  ChatroomMgr listening for Requests...")
	var newChat *ChatroomBroker
	for {
		select {
		case conn := <-cm.newConnCh:
			go cm.HandleNewConnectPrompt(conn)
		case unsubEvent := <-cm.unsubCh:
			log.Printf("User%d left chat '%s'\n", unsubEvent.UserId, cm.chatroomIndex[unsubEvent.ChatId].ChatroomName)
			delete(cm.chatroomIndex[unsubEvent.ChatId].subs, unsubEvent.UserId)
			go cm.MoveExistingConnectPrompt(unsubEvent.UserId)
		case newChatName := <-cm.createChatCh:
			chatId := len(cm.chatroomIndex) + 1
			log.Printf("Creating new chat: %s:%d\n", newChatName, chatId)
			newChat = cm.AddNewChatroom(newChatName, chatId)
			go newChat.Start()
		case <-cm.quitCh:
			log.Println("ChatroomMgr shutting down...")
			return
		}
	}
}

func (cm *ChatroomManager) HandleNewConnectPrompt(conn net.Conn) {
	userId := rand.Intn(1000)
	ch := NewConnHandler(userId, conn, cm.unsubCh, cm.quitCh)
	ch.conn.Write([]byte(fmt.Sprintf(LOGIN_PROMPT, userId)))
	ch.conn.Write([]byte(cm.ReadAllChatrooms()))
	ch.conn.Write([]byte("Which chatroom would you like? "))
	var msg string
	for {
		switch msg = ch.readFromConnOnce(); msg {
		default:
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
		case "x":
			ch.conn.Write([]byte("\nWhat would you like to name the new chat? "))
			newChatName := ch.readFromConnOnce()
			log.Println("Creating new chatroom named", newChatName)
			cm.createChatCh <- newChatName
			time.Sleep(time.Millisecond * 10)
			ch.conn.Write([]byte(cm.ReadAllChatrooms()))
			ch.conn.Write([]byte("Which chatroom would you like? "))
		case "r":
			ch.conn.Write([]byte(cm.ReadAllChatrooms()))
			ch.conn.Write([]byte("Which chatroom would you like? "))
		case "$exit":
			ch.conn.Write([]byte("\nThanks for coming!\n"))
			delete(cm.activeConns, userId)
			ch.conn.Close()
			return
		case "":
			return // received SIGTERM
		}
	}
}

func (cm *ChatroomManager) MoveExistingConnectPrompt(userId int) {
	ch := cm.activeConns[userId]
	ch.conn.Write([]byte(cm.ReadAllChatrooms()))
	ch.conn.Write([]byte("Which chatroom to change to? "))
	var msg string
	for {
		switch msg = ch.readFromConnOnce(); msg {
		default:
			chosenChatId, err := strconv.Atoi(msg)
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
		case "x":
			ch.conn.Write([]byte("\nWhat would you like to name the new chat? "))
			newChatName := ch.readFromConnOnce()
			log.Println("Creating new chatroom named", newChatName)
			cm.createChatCh <- newChatName
			time.Sleep(time.Millisecond * 10) // createChatCh needs a tiny bit of time to update
			ch.conn.Write([]byte(cm.ReadAllChatrooms()))
			ch.conn.Write([]byte("Which chatroom would you like? "))
		case "r":
			ch.conn.Write([]byte(cm.ReadAllChatrooms()))
			ch.conn.Write([]byte("Which chatroom would you like? "))
		case "$exit":
			ch.conn.Write([]byte("\nThanks for coming!\n"))
			delete(cm.activeConns, userId)
			ch.conn.Close()
			return
		case "":
			return // received SIGTERM
		}
	}
}

func (cm *ChatroomManager) AddNewChatroom(newChatName string, chatId int) *ChatroomBroker {
	cm.rwLock.Lock()
	defer cm.rwLock.Unlock()
	if _, ok := cm.chatroomIndex[chatId]; ok {
		log.Println("ERR: chatroom # already exists")
		return nil
	}
	for _, c := range cm.chatroomIndex {
		if newChatName == c.ChatroomName {
			log.Println("ERR: chatroom name already exists")
			return nil
		}
	}
	cm.chatroomIndex[chatId] = NewChatroomBroker(newChatName, chatId, cm.quitCh)
	return cm.chatroomIndex[chatId]
}

func (cm *ChatroomManager) ReadAllChatrooms() string {
	cm.rwLock.Lock()
	defer cm.rwLock.Unlock()
	reportString := "\n# Available chats:\n[\n"
	for _, chatroom := range cm.chatroomIndex {
		reportString += fmt.Sprintf("  -> [%d] %s\n", chatroom.ChatId, chatroom.ChatroomName)
	}
	reportString += "  -> [x] Create new chatroom\n"
	reportString += "  -> [r] Refresh chat list\n"
	reportString += "]\n"
	return reportString
}

func (cm *ChatroomManager) DoesChatroomExist(chatId int) bool {
	cm.rwLock.Lock()
	defer cm.rwLock.Unlock()
	_, ok := cm.chatroomIndex[chatId]
	return ok
}
