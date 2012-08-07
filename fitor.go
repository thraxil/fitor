package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"code.google.com/p/go.net/websocket"
	"fmt"
	irc "github.com/fluffle/goirc/client"
	"net/http"
	"strconv"
	"strings"
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
	token := ws.Request().URL.Query().Get("token")

	// token will look something like this:
	// anp8:1344361884:667494:127.0.0.1:306233f64522f1f970fc62fb3cf2d7320c899851
	parts := strings.Split(token, ":")
	if len(parts) != 5 {
		fmt.Println("invalid token")
		return
	}
	// their UNI
	uni := parts[0]
	// UNIX timestamp
	now, err := strconv.Atoi(parts[1])
	if err != nil {
		fmt.Printf("invalid timestamp in token")
		return
	}
	// a random salt 
	salt := parts[2]
	ip_address := parts[3]
	// the hmac of those parts with our shared secret
	hmc := parts[4]

	// make sure we're within a 60 second window
	current_time := time.Now()
	token_time := time.Unix(int64(now),0)
	if current_time.Sub(token_time) > time.Duration(60 * time.Second) {
		fmt.Printf("stale token\n")
		fmt.Printf("%s %s\n", current_time, token_time)
		return
	}
	// TODO: check that their ip address matches

	// check that the HMAC matches
	h := hmac.New(
		sha1.New,
		[]byte("6f1d916c-7761-4874-8d5b-8f8f93d20bf2"))
	h.Write([]byte(fmt.Sprintf("%s:%d:%s:%s",uni,now,salt,ip_address)))
	sum := fmt.Sprintf("%x", h.Sum(nil))
	if sum != hmc {
		fmt.Println("token HMAC doesn't match")
		return
	}
	
	onlineUser := &OnlineUser{
		Connection: ws,
		Nick:       uni,
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
