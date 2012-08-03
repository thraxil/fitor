$(function() {
	  var conn;
	  var log = $("#log");

	  function appendLog(msg) {
        var d = log[0]
        var doScroll = d.scrollTop == d.scrollHeight - d.clientHeight;
        msg.appendTo(log)
        if (doScroll) {
            d.scrollTop = d.scrollHeight - d.clientHeight;
        }
	  }

	  if (window["WebSocket"]) {
    	  conn = new WebSocket("ws://behemoth.ccnmtl.columbia.edu:5050/socket/");
	      conn.onclose = function(evt) {
	          appendLog($("<div><b>Connection closed.</b></div>"))
	      }
	      conn.onmessage = function(evt) {

					  var data = JSON.parse(evt.data);
					  var entry = $("<div/>");
            entry.addClass("row");
            var d = new Date(Date.parse(data.Time));
            entry.append("<div class='span2 timestamp'>" + d.getHours() + ":" + d.getMinutes() + "</div>");
					  entry.append("<div class='span1 nick'>" + data.Nick + "</div>");
					  entry.append("<div class='span9 ircmessage'>" + data.Content + "</div>");
	          appendLog(entry);
	      }
	  } else {
	      appendLog($("<div><b>Your browser does not support WebSockets.</b></div>"))
	  }
});

