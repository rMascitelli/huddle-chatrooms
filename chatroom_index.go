package main

import (
	"errors"
	"fmt"
	"sync"
)

// Simple map of chatrooms that will be globally available for ConnHandlers to access
// With Mutex to protect reads
type ChatroomIndex struct {
	chatrooms map[string]int
	rwLock    sync.Mutex
}

func NewChatroomIndex() ChatroomIndex {
	ci := ChatroomIndex{
		chatrooms: make(map[string]int),
		rwLock:    sync.Mutex{},
	}
	ci.CreateDefaultChatrooms()
	return ci
}

func (ci *ChatroomIndex) AddNewChatroom(chatroomName string) error {
	ci.rwLock.Lock()
	defer ci.rwLock.Unlock()
	if _, ok := ci.chatrooms[chatroomName]; ok {
		return errors.New("chatroom already exists")
	}
	return nil
}

func (ci *ChatroomIndex) ReadAllChatrooms() string {
	ci.rwLock.Lock()
	defer ci.rwLock.Unlock()
	reportString := "\n# Available chats:\n[\n"
	for name, _ := range ci.chatrooms {
		reportString += fmt.Sprintf("  -> %s\n", name)
	}
	reportString += "]\n"
	return reportString
}

func (ci *ChatroomIndex) CreateDefaultChatrooms() {
	ci.chatrooms["DefaultTestChat1"] = 1
	ci.chatrooms["DefaultTestChat2"] = 2
	ci.chatrooms["DefaultTestChat3"] = 3
	ci.chatrooms["SpookyMonsterChat"] = 4
}
