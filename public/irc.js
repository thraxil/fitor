$(function() {
	  var conn;
	  var log = $("#log");

    var currentRefresh = 1000;
    var defaultRefresh = 1000;
    var maxRefresh = 1000 * 5 * 60; // 5 minutes

    window.nick = "";

    var requestFailed = function(evt) {
        // circuit breaker pattern for failed requests
        // to ease up on the server when it's having trouble
        currentRefresh = 2 * currentRefresh; // double the refresh time
        if (currentRefresh > maxRefresh) {
            currentRefresh = maxRefresh;
        }
        appendLog($("<div class='alert'><b>Connection closed. trying again in " + currentRefresh/1000 + " seconds</b></div>"));
        setTimeout(connectSocket,currentRefresh);
    };

    var connectSocket = function() {
        while (window.nick === null || window.nick === "") {
            window.nick = prompt("Please enter a nick");
        }
    	  conn = new WebSocket("ws://behemoth.ccnmtl.columbia.edu:5050/socket/?nick=" + window.nick);
	      conn.onclose = requestFailed;
	      conn.onmessage = onMessage;
        conn.onopen = function (evt) {
            currentRefresh = defaultRefresh;
            appendLog($("<div class='alert alert-info'><b>Connected to server.</b></div>"));
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
					  entry.append("<div class='span2 nick'>&lt;" + data.Nick + "&gt;</div>");
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


    $("#msg_form").submit(function() {
				var msg = $("#appendedPrependedInput");
				if (!conn) {
			  		return false;
				}
				if (!msg.val()) {
			  		return false;
				}
				conn.send(msg.val());
				msg.val("");
				return false
			});

	  if (window["WebSocket"]) {
        connectSocket();
	  } else {
	      appendLog($("<div><b>Your browser does not support WebSockets.</b></div>"))
	  }
});

