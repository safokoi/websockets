package main

import (
	"fmt"
	"net/http"

	"golang.org/x/net/websocket"
)

var port = ":8000"

type client struct {
	name    string
	channel chan interface{}
	status  string
}

var clients = make(map[*websocket.Conn]client)

var announcement = make(chan client)

func main() {
	fmt.Printf("Starting server on port %s\n", port)
	http.Handle("/ws", websocket.Handler(webSocketHandler))
	http.Handle("/", http.FileServer(http.Dir("./static")))

	go manager()

	if err := http.ListenAndServe(port, nil); err != nil {
		panic("ListenAndServer:" + err.Error())
	}
}

func manager() {
	for {
		client := <-announcement
		for k := range clients {
			websocket.JSON.Send(k, fmt.Sprintf("'%v' has %v ", client.name, client.status))
		}
	}
}

func webSocketHandler(ws *websocket.Conn) {
	defer func() {
		ws.Close()
		client, ok := clients[ws]
		if ok {
			delete(clients, ws)
			client.status = "left"
			announcement <- client
		}
		fmt.Printf("Connection closed\n")
	}()

	var name string
	err := websocket.Message.Send(ws, "Please provide the desired name")
	if err != nil {
		return
	}

	err = websocket.Message.Receive(ws, &name)
	if err != nil {
		return
	}

	for k := range clients {
		switch clients[k].name {
		case name:
			websocket.Message.Send(ws, fmt.Sprintf("Sorry, the name '%v' is already taken, bye\n", name))
			return
		}
	}

	client := client{name, make(chan interface{}), "joined"}
	clients[ws] = client

	announcement <- client

	fmt.Printf("Client with the name '%v' has joined\n", client.name)
	fmt.Printf("Overall connected clients: %v\n", len(clients))

	go func() {
	out:
		for {
			select {
			case message := <-client.channel:
				for k, v := range clients {
					websocket.JSON.Send(k, message)
					fmt.Printf("Message '%v' was sent to '%v' \n", message, v.name)
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
		client.channel <- client.name + ": " + message
	}

}
