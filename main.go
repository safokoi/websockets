package main

import (
	"fmt"
	"net/http"
	"sync"

	"golang.org/x/net/websocket"
)

var port = ":8000"

type client struct {
	username string
	channel  chan interface{}
	status   string
}

type clients struct {
	sync.Mutex
	list map[*websocket.Conn]client
}

var roomies = clients{list: make(map[*websocket.Conn]client)}

var announcement = make(chan client)

func main() {
	fmt.Printf("Starting server on port %s\n", port)
	http.Handle("/ws", websocket.Handler(webSocketHandler))
	http.Handle("/", http.FileServer(http.Dir("./static")))

	go announcer()

	if err := http.ListenAndServe(port, nil); err != nil {
		panic("ListenAndServer:" + err.Error())
	}
}

func announcer() {
	for {
		roomie := <-announcement
		for k := range roomies.list {
			websocket.JSON.Send(k, fmt.Sprintf("'%v' has %v ", roomie.username, roomie.status))
		}
	}
}

func webSocketHandler(ws *websocket.Conn) {
	defer func() {
		ws.Close()
		roomie, ok := roomies.list[ws]
		if ok {
			roomies.Mutex.Lock()
			delete(roomies.list, ws)
			roomies.Mutex.Unlock()
			roomie.status = "left"
			announcement <- roomie
		}
		fmt.Printf("Connection closed\n")
	}()

	fmt.Printf("websocket.Conn is %v\n", ws)
	var name string
	err := websocket.Message.Send(ws, "Please provide the desired name")
	if err != nil {
		return
	}

	err = websocket.Message.Receive(ws, &name)
	if err != nil {
		return
	}

	for k := range roomies.list {
		switch roomies.list[k].username {
		case name:
			websocket.Message.Send(ws, fmt.Sprintf("Sorry, the name '%v' is already taken, bye\n", name))
			return
		}
	}

	roomie := client{name, make(chan interface{}), "joined"}
	roomies.Mutex.Lock()
	roomies.list[ws] = roomie
	roomies.Mutex.Unlock()

	announcement <- roomie

	fmt.Printf("Client with the name '%v' has joined\n", roomie.username)
	fmt.Printf("Overall connected clients: %v\n", len(roomies.list))

	go func() {
	out:
		for {
			select {
			case message := <-roomie.channel:
				for k, v := range roomies.list {
					websocket.JSON.Send(k, message)
					fmt.Printf("Message '%v' was sent to '%v' \n", message, v.username)
				}
			case <-ws.Request().Context().Done():
				fmt.Printf("Goroutine is killed\n")
				break out
			}
		}
	}()

	for {
		var message string
		err := websocket.Message.Receive(ws, &message)
		if err != nil {
			break
		}
		fmt.Printf("Message '%v' was received\n", message)
		roomie.channel <- roomie.username + ": " + message
	}

}
