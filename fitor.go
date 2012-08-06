package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	"net/http"
	"time"
)

type room struct {
	Users     map[*OnlineUser]bool
	Broadcast chan Message
	Incoming  chan IncomingMessage
}

type Message struct {
	Time    time.Time
	Nick    string
	Content string
}

type IncomingMessage struct {
	Type    string
	Content string
	Nick    string
}

var runningRoom *room = &room{}

func (r *room) run() {
	for b := range r.Broadcast {
		for u := range r.Users {
			u.Send <- b
		}
	}
}

func (r *room) SendLine(line Message) {
	r.Broadcast <- line
}

func InitRoom() {
	runningRoom = &room{
		Users:     make(map[*OnlineUser]bool),
		Broadcast: make(chan Message),
		Incoming:  make(chan IncomingMessage),
	}
	go runningRoom.run()
}

type OnlineUser struct {
	Connection *websocket.Conn
	Nick       string
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
		runningRoom.Incoming <- IncomingMessage{"msg", content, this.Nick}
		// need to echo back to ourself
		msg := Message{time.Now(), this.Nick, content}
		runningRoom.SendLine(msg)
	}
}

func BuildConnection(ws *websocket.Conn) {
	nick := ws.Request().URL.Query().Get("nick")
	onlineUser := &OnlineUser{
		Connection: ws,
		Nick:       nick,
		Send:       make(chan Message, 256),
	}
	runningRoom.Users[onlineUser] = true
	go onlineUser.PushToClient()
	runningRoom.Incoming <- IncomingMessage{"notice", "joined as web user", nick}
	onlineUser.PullFromClient()
	runningRoom.Incoming <- IncomingMessage{"notice", "web user disconnected", nick}
	delete(runningRoom.Users,onlineUser)
}

func main() {
	InitRoom()

	c := irc.SimpleClient("gobot")

	// Add handlers to do things here!
	c.AddHandler("connected", func(conn *irc.Conn, line *irc.Line) {
		conn.Join("#ccnmtl")
		go func() {
			for msg := range runningRoom.Incoming {
				if msg.Type == "msg" {
					conn.Privmsg("#ccnmtl", msg.Nick + ": " + msg.Content)
				} else if msg.Type == "notice" {
					conn.Notice("#ccnmtl", msg.Nick + ": " + msg.Content)
				}
			}
		}()
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
</head>
<body>
					 <h1>Our IRC channel</h1>
					 <p>(you may have to wait for someone to post something)</p>

	<div id="log"></div>

<div id="input-box" class="span8">
<form id="msg_form" class="form-horizontal post-form">
<div class="input-append">
<input class="span7" id="appendedPrependedInput" size="16" type="text"><input type="submit" class="btn" value="Post" />
</div>
</form>
</div>

		<script src="/public/irc.js"></script>
<script src="/public/javascripts/bootstrap.js"></script>
</body>
</html>
`
	w.Write([]byte(page))
}
