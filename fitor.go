package main

/* just an IRC <-> 0MQ bridge */

import (
	"encoding/json"
	"errors"
	"fmt"
	zmq "github.com/alecthomas/gozmq"
	irc "github.com/fluffle/goirc/client"
)

var PUB_KEY = "gobot.fitor"
var SUB_KEY = "gobot"
var NICK = "gobot"
var CHANNEL = "#ccnmtl"

var REQ_SOCKET = "tcp://localhost:5555"
var SUB_SOCKET = "tcp://localhost:5556"

type Message struct {
	MessageType string `json:"message_type"`
	Nick        string `json:"nick"`
	Content     string `json:"content"`
}

// how we route zmq messages around
type envelope struct {
	Address string `json:"address"`
	Content string `json:"content"`
}

func startswith(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}

func receiveZmqMessage(subsocket zmq.Socket, m *Message) error {
	// using zmq multi-part messages which will arrive
	// in pairs. the first of which we don't care about so we discard.
	address, _ := subsocket.Recv(0)
	content, _ := subsocket.Recv(0)
	if startswith(string(address), PUB_KEY) {
		// it's one that we sent out, so ignore it
		return errors.New("do not echo my own messages")
	}
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

// send a message to the zmq REQ socket
func sendMessage(reqsocket zmq.Socket, m Message) {
	var address = PUB_KEY + "." + m.Nick
	b, _ := json.Marshal(m)
	var content = b
	env := envelope{address, string(content)}
	e, _ := json.Marshal(env)
	reqsocket.Send([]byte(e), 0)
	// wait for a reply
	reqsocket.Recv(0)
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
	reqsocket, _ := context.NewSocket(zmq.REQ)
	subsocket, _ := context.NewSocket(zmq.SUB)
	defer context.Close()
	defer reqsocket.Close()
	defer subsocket.Close()
	reqsocket.Connect(REQ_SOCKET)
	subsocket.SetSockOptString(zmq.SUBSCRIBE, SUB_KEY)
	subsocket.Connect(SUB_SOCKET)

	// configure our IRC client
	c := irc.SimpleClient(NICK)

	// most of the functionality is arranged by adding handlers
	c.AddHandler("connected", func(conn *irc.Conn, line *irc.Line) {
		conn.Join(CHANNEL)
		sendMessage(reqsocket, statusMessage("joined "+CHANNEL))
		// spawn a goroutine that will do the ZMQ -> IRC bridge
		go zmqToIrcLoop(conn, subsocket)
	})

	quit := make(chan bool)
	c.AddHandler("disconnected", func(conn *irc.Conn, line *irc.Line) {
		sendMessage(reqsocket, statusMessage("disconnected"))
		quit <- true
	})
	// this is the handler that gets triggered whenever someone posts
	// in the channel
	c.AddHandler("PRIVMSG", func(conn *irc.Conn, line *irc.Line) {
		// forward messages from IRC -> zmq PUB socket
		if line.Nick != NICK {
			sendMessage(reqsocket, Message{"message", line.Nick, line.Args[1]})
		}
	})

	// Now we can connect
	if err := c.Connect("irc.freenode.net"); err != nil {
		sendMessage(reqsocket, statusMessage("error connecting: "+err.Error()))
		fmt.Printf("Connection error: %s\n", err.Error())
	}

	// Wait for disconnect
	<-quit
}
