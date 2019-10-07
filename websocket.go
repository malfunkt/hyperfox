package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
	//	"github.com/malfunkt/hyperfox/lib/plugins/capture"
)

var (
	wsClients = map[*websocket.Conn]struct{}{}
	wsMu      = sync.Mutex{}
)

var wsUpgrader = websocket.Upgrader{} // use default options

func init() {
	wsUpgrader.CheckOrigin = func(r *http.Request) bool { return true }
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}

	go wsAddConn(conn)
}

func wsAddConn(conn *websocket.Conn) {
	wsMu.Lock()
	defer wsMu.Unlock()

	wsClients[conn] = struct{}{}
	go wsReadMessages(conn)

	wsSendMessage(conn, nil)
}

func wsReadMessages(conn *websocket.Conn) error {
	for {
		reply := map[string]string{}

		if err := websocket.ReadJSON(conn, &reply); err != nil {
			fmt.Println("Can't receive: %v", err)
			defer wsRemoveConn(conn)
			return err
		}
		log.Printf("reply: %v", reply)
	}
	return nil
}

func wsBroadcast(message interface{}) error {
	wsMu.Lock()

	for conn := range wsClients {
		if err := wsSendMessage(conn, message); err != nil {
			log.Printf("wsSendMessage: ", err)
			defer wsRemoveConn(conn)
		}
	}

	wsMu.Unlock()
	return nil
}

func wsRemoveConn(conn *websocket.Conn) {
	wsMu.Lock()
	defer wsMu.Unlock()

	if _, ok := wsClients[conn]; ok {
		conn.Close()
		delete(wsClients, conn)
	}
}

func wsSendMessage(conn *websocket.Conn, message interface{}) error {
	return websocket.WriteJSON(conn, message)
}
