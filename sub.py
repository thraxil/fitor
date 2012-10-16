"""
A very simple python script that listens in
on the ZMQ messages that the IRC client broadcasts
and prints them out to the console.
"""

import sys
import zmq
from simplejson import loads

PUB_KEY = "gobot"
PUB_SOCKET = "tcp://localhost:5556"

context = zmq.Context()
socket = context.socket(zmq.SUB)
socket.connect (PUB_SOCKET)
socket.setsockopt(zmq.SUBSCRIBE, PUB_KEY)

while True:
    [address, contents] = socket.recv_multipart()
    message = loads(contents)
    if message['message_type'] == "status":
        print "*** %s" % message['content']
    else:
        print "%s: %s" % (message['nick'], message['content'])

