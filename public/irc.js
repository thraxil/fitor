$(function() {
	  var conn;
	  var log = $("#log");

    var currentRefresh = 1000;
    var defaultRefresh = 1000;
    var maxRefresh = 1000 * 5 * 60; // 5 minutes

    var requestFailed = function(evt) {
        // circuit breaker pattern for failed requests
        // to ease up on the server when it's having trouble
        currentRefresh = 2 * currentRefresh; // double the refresh time
        if (currentRefresh > maxRefresh) {
            currentRefresh = maxRefresh;
        }
        appendLog($("<div><b>Connection closed. trying again in " + currentRefresh/1000 + " seconds</b></div>"));
        setTimeout(connectSocket,currentRefresh);
    };

    var connectSocket = function() {
    	  conn = new WebSocket("ws://behemoth.ccnmtl.columbia.edu:5050/socket/");
	      conn.onclose = requestFailed;
	      conn.onmessage = onMessage;
        conn.onopen = function (evt) {
            currentRefresh = defaultRefresh;
            appendLog($("<div><b>Connected to server.</b></div>"));
        };
    };

    var onMessage = function (evt) {
					  var data = JSON.parse(evt.data);
					  var entry = $("<div/>");
            entry.addClass("row");
            var d = new Date(Date.parse(data.Time));
            var hours = d.getHours()
	          var minutes = d.getMinutes()

	          if (minutes < 10) {
	              minutes = "0" + minutes
            }
            entry.append("<div class='span1 timestamp'>" + hours + ":" + minutes + "</div>");
					  entry.append("<div class='span1 nick'>&lt;" + data.Nick + "&gt;</div>");
					  entry.append("<div class='span10 ircmessage'>" + data.Content + "</div>");
	          appendLog(entry);
    }

	  function appendLog(msg) {
        var d = log[0]
        var doScroll = d.scrollTop == d.scrollHeight - d.clientHeight;
        msg.appendTo(log)
        if (doScroll) {
            d.scrollTop = d.scrollHeight - d.clientHeight;
        }
	  }

	  if (window["WebSocket"]) {
        connectSocket();
	  } else {
	      appendLog($("<div><b>Your browser does not support WebSockets.</b></div>"))
	  }
});

