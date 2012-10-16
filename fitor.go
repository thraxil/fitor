package main

/* just an IRC <-> 0MQ bridge */

import (
	"encoding/json"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	irc "github.com/fluffle/goirc/client"
)

var PUB_KEY = "gobot"
var NICK = "gobot"
var CHANNEL = "#ccnmtl"

var PUB_SOCKET = "tcp://*:5556"
var SUB_SOCKET = "tcp://localhost:5557"

type Message struct {
	MessageType string `json:"message_type"`
	Nick        string `json:"nick"`
	Content     string `json:"content"`
}

func receiveZmqMessage(subsocket zmq.Socket, m *Message) error {
	// using zmq multi-part messages which will arrive
	// in pairs. the first of which we don't care about so we discard.
	_, _ = subsocket.Recv(0)
	content, _ := subsocket.Recv(0)
	return json.Unmarshal([]byte(content), m)
}

// listen on a zmq SUB socket and forward messages
// to the IRC connection
func zmqToIrcLoop(conn *irc.Conn, subsocket zmq.Socket) {
	var m Message
	for {
		err := receiveZmqMessage(subsocket, &m)
		if err != nil {
			// just ignore any invalid messages
			continue
		}
		if m.MessageType != "message" {
			conn.Privmsg(CHANNEL, "["+m.Content+"]")
		} else {
			ircmessage := fmt.Sprintf("%s: %s", m.Nick, m.Content)
			conn.Privmsg(CHANNEL, ircmessage)
		}
	}
}

// send a message to the zmq PUB socket
func sendMessage(pubsocket zmq.Socket, m Message) {
	b, _ := json.Marshal(m)
	pubsocket.SendMultipart([][]byte{[]byte(PUB_KEY), b}, 0)
}

func statusMessage(content string) Message {
	return Message{
		MessageType: "status",
		Nick:        NICK,
		Content:     content,
	}
}

func main() {
	// prepare our zmq sockets
	context, _ := zmq.NewContext()
	pubsocket, _ := context.NewSocket(zmq.PUB)
	subsocket, _ := context.NewSocket(zmq.SUB)
	defer context.Close()
	defer pubsocket.Close()
	defer subsocket.Close()
	pubsocket.Bind(PUB_SOCKET)
	subsocket.SetSockOptString(zmq.SUBSCRIBE, PUB_KEY)
	subsocket.Connect(SUB_SOCKET)

	// configure our IRC client
	c := irc.SimpleClient(NICK)

	// most of the functionality is arranged by adding handlers
	c.AddHandler("connected", func(conn *irc.Conn, line *irc.Line) {
		conn.Join(CHANNEL)
		sendMessage(pubsocket, statusMessage("joined "+CHANNEL))
		// spawn a goroutine that will do the ZMQ -> IRC bridge
		go zmqToIrcLoop(conn, subsocket)
	})

	quit := make(chan bool)
	c.AddHandler("disconnected", func(conn *irc.Conn, line *irc.Line) {
		sendMessage(pubsocket, statusMessage("disconnected"))
		quit <- true
	})
	// this is the handler that gets triggered whenever someone posts
	// in the channel
	c.AddHandler("PRIVMSG", func(conn *irc.Conn, line *irc.Line) {
		// forward messages from IRC -> zmq PUB socket
		sendMessage(pubsocket, Message{"message", line.Nick, line.Args[1]})
	})

	// Now we can connect
	if err := c.Connect("irc.freenode.net"); err != nil {
		sendMessage(pubsocket, statusMessage("error connecting: "+err.Error()))
		fmt.Printf("Connection error: %s\n", err.Error())
	}

	// Wait for disconnect
	<-quit
}
