package main

import (
	"fmt"
	"log"
	"sync"
)

// PubSub manager, to receive and broadcast messages to one chatroom
type ChatroomBroker struct {
	publishCh chan ChatMessage // Rx messages from all connHandlers
	subCh     chan ConnHandler // Rx from ChatroomMgr about new subscriber
	subs      map[int]*ConnHandler

	wg     *sync.WaitGroup
	quitCh chan struct{}
}

func NewChatroomBroker(quitCh chan struct{}, subCh chan ConnHandler, wg *sync.WaitGroup) *ChatroomBroker {
	return &ChatroomBroker{
		publishCh: make(chan ChatMessage),
		quitCh:    quitCh,
		subs:      make(map[int]*ConnHandler),
		wg:        wg,
		subCh:     subCh,
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
	log.Println("ChatroomBroker starting...")
	for {
		select {
		case msg := <-cb.publishCh:
			// Broadcast msg to all subs
			for _, sub := range cb.subs {
				DebugPrint(fmt.Sprintf("%v to %d", msg.Payload, sub.UserId))
				sub.MsgCh <- msg
			}
		case ch := <-cb.subCh:
			ch.PublishCh = cb.publishCh
			go ch.startHandlingConn()
			log.Printf("User%d joined the Chatroom\n", ch.UserId)
			cb.subs[ch.UserId] = &ch
			cb.PrintSubs()
		}
	}
}

func (cb *ChatroomBroker) Unsubscribe(userId int) {
	fmt.Println("Deleting user", userId)
	delete(cb.subs, userId)
}
