package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
)

// PubSub manager, to receive and broadcast messages to one chatroom
type ChatroomBroker struct {
	publishCh chan ChatMessage     // Mutual ch to receive messages from ConnHandlers
	quitCh    chan struct{}        // Mutual ch to listen for close()
	subs      map[int]*ConnHandler // Keeping record of ConnHandlers
	wg        *sync.WaitGroup

	subCh   chan net.Conn // Use conn to create a ConnHandler and add to subsmap
	unsubCh chan int      // Use Userid to find and delete sub from submap
}

func NewChatroomBroker(quitCh chan struct{}, subCh chan net.Conn, unsubCh chan int, wg *sync.WaitGroup) *ChatroomBroker {
	return &ChatroomBroker{
		publishCh: make(chan ChatMessage),
		quitCh:    quitCh,
		subs:      make(map[int]*ConnHandler),
		wg:        wg,
		subCh:     subCh,
		unsubCh:   unsubCh,
	}
}

func (cb *ChatroomBroker) PrintSubs() {
	fmt.Printf("[")
	for _, c := range cb.subs {
		fmt.Printf("%d, ", c.Userid)
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
				DebugPrint(fmt.Sprintf("%v to %d", msg.Payload, sub.Userid))
				sub.MsgCh <- msg
			}
		case conn := <-cb.subCh:
			userId := rand.Intn(1000)
			ch := NewConnHandler(userId, conn, cb.wg, cb.publishCh, cb.quitCh)
			go ch.startHandlingConn()
			log.Printf("User%d joined the Chatroom\n", userId)
			cb.subs[userId] = ch
			cb.PrintSubs()
		}
	}
}
