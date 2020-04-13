package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	wsClients = map[*websocket.Conn]struct{}{}
	wsMu      = sync.Mutex{}
	wsDebug   = false
)

var wsUpgrader = websocket.Upgrader{} // use default options

func init() {
	wsUpgrader.CheckOrigin = func(r *http.Request) bool { return true }
}

func liveHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	go wsAddConn(conn)
}

func wsAddConn(conn *websocket.Conn) {
	wsMu.Lock()
	defer wsMu.Unlock()

	wsClients[conn] = struct{}{}
	go func() {
		if err := wsReadMessages(conn); err != nil {
			if wsDebug {
				log.Print("ws read: ", err)
			}
		}
	}()

	if err := wsSendMessage(conn, nil); err != nil {
		if wsDebug {
			log.Print("ws send: ", err)
		}
	}
}

func wsReadMessages(conn *websocket.Conn) error {
	for {
		reply := map[string]string{}

		if err := conn.ReadJSON(&reply); err != nil {
			defer wsRemoveConn(conn)
			return err
		}
	}
}

func wsBroadcast(message interface{}) error {
	wsMu.Lock()

	for conn := range wsClients {
		if err := wsSendMessage(conn, message); err != nil {
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
	return conn.WriteJSON(message)
}
