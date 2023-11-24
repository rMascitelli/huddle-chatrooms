package main

import (
	"fmt"
	"log"
)

// PubSub manager, to receive and broadcast messages to one chatroom
type ChatroomBroker struct {
	ChatroomName string
	ChatId       int
	publishCh    chan ChatMessage  // Rx messages from all connHandlers
	subCh        chan *ConnHandler // Rx from ChatroomMgr about new subscriber
	subs         map[int]*ConnHandler

	quitCh chan struct{}
}

func NewChatroomBroker(chatroomName string, chatId int, quitCh chan struct{}, subCh chan *ConnHandler) *ChatroomBroker {
	DebugPrint(fmt.Sprintf("NewChatroomBroker: Creating chat %s:%d\n", chatroomName, chatId))
	return &ChatroomBroker{
		ChatroomName: chatroomName,
		ChatId:       chatId,
		publishCh:    make(chan ChatMessage),
		quitCh:       quitCh,
		subs:         make(map[int]*ConnHandler),
		subCh:        subCh,
	}
}

func (cb *ChatroomBroker) PrintSubs() {
	fmt.Printf("[")
	for _, c := range cb.subs {
		fmt.Printf("%d, ", c.UserId)
	}
	fmt.Printf("]\n")
}

func (cb *ChatroomBroker) Start() {
	DebugPrint(fmt.Sprintf("Chat '%s:%d' starting...\n", cb.ChatroomName, cb.ChatId))
	for {
		select {
		case msg := <-cb.publishCh:
			// Broadcast msg to all subs
			for _, sub := range cb.subs {
				DebugPrint(fmt.Sprintf("%v to %d", msg.Payload, sub.UserId))
				sub.MsgCh <- msg
			}
		case ch := <-cb.subCh:
			log.Printf("CB%s rcvd sub request\n", cb.ChatroomName)
			ch.PublishCh = cb.publishCh
			go ch.startHandlingConn()
			log.Printf("User%d joined the Chatroom\n", ch.UserId)
			cb.subs[ch.UserId] = ch
			cb.PrintSubs()
		}
	}
}

func (cb *ChatroomBroker) Unsubscribe(userId int) {
	fmt.Printf("Removing user%d from chat\n", userId)
	delete(cb.subs, userId)
}
