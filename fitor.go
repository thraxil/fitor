package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	"net/http"
	"time"
)

type room struct {
	Users     []*OnlineUser
	Broadcast chan Message
}

type Message struct {
	Time    time.Time
	Nick    string
	Content string
}

var runningRoom *room = &room{}

func (r *room) run() {
	for b := range r.Broadcast {
		for _, u := range r.Users {
			u.Send <- b
		}
	}
}

func (r *room) SendLine(line Message) {
	r.Broadcast <- line
}

func InitRoom() {
	runningRoom = &room{
		Users:     make([]*OnlineUser, 0),
		Broadcast: make(chan Message),
	}
	go runningRoom.run()
}

type OnlineUser struct {
	Connection *websocket.Conn
	Send       chan Message
}

func (this *OnlineUser) PushToClient() {
	for b := range this.Send {
		err := websocket.JSON.Send(this.Connection, b)
		if err != nil {
			break
		}
	}
}

func (this *OnlineUser) PullFromClient() {
	for {
		var content string
		err := websocket.Message.Receive(this.Connection, &content)

		if err != nil {
			return
		}
		// don't actually do anything here. just keeping it open
	}
}

func BuildConnection(ws *websocket.Conn) {
	onlineUser := &OnlineUser{
		Connection: ws,
		Send:       make(chan Message, 256),
	}
	runningRoom.Users = append(runningRoom.Users, onlineUser)
	go onlineUser.PushToClient()
	onlineUser.PullFromClient()

}

func main() {
	InitRoom()

	c := irc.SimpleClient("gobot")

	// Add handlers to do things here!
	c.AddHandler("connected", func(conn *irc.Conn, line *irc.Line) {
		conn.Join("#ccnmtl")
	})
	quit := make(chan bool)
	c.AddHandler("disconnected", func(conn *irc.Conn, line *irc.Line) {
		quit <- true
	})
	c.AddHandler("PRIVMSG", func(conn *irc.Conn, line *irc.Line) {
		msg := Message{line.Time, line.Nick, line.Args[1]}
		runningRoom.SendLine(msg)
	})

	// Tell client to connect
	if err := c.Connect("irc.freenode.net"); err != nil {
		fmt.Printf("Connection error: %s\n", err.Error())
	}

	http.HandleFunc("/", Home)
	http.Handle("/socket/", websocket.Handler(BuildConnection))
	http.HandleFunc("/public/", assetsHandler)
	err := http.ListenAndServe(":5050", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}

	// Wait for disconnect
	<-quit
}

func assetsHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, r.URL.Path[len("/"):])
}

func Home(w http.ResponseWriter, r *http.Request) {
	page := `
<html>
<head><title>IRC websocket test</title>
    <link href="/public/stylesheets/bootstrap.css" rel="stylesheet">
    <link href="/public/stylesheets/main.css" rel="stylesheet">
    <script src="/public/javascripts/jquery-1.7.2.min.js"></script>
    <style>
      body {
        padding-top: 60px; /* 60px to make the container go all the way to the bottom of the topbar */
      }
    </style>
</head>
<body>
					 <h1>Our IRC channel</h1>
					 <p>(you may have to wait for someone to post something)</p>

	<div id="log"></div>

		<script src="/public/irc.js"></script>
<script src="/public/javascripts/bootstrap.js"></script>
</body>
</html>
`
	w.Write([]byte(page))
}
