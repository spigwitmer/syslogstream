function GetWebsocket(url) {
	let socket = new WebSocket(url);

	socket.onmessage = function(event) {
	  BroadcastLine(event.data);
	};

	socket.onerror = function(error) {
      console.debug(data);
	  BroadcastLine('ERROR: ' + error.data);
	};

	return socket;
}

var logsTxt = document.getElementById('logs');
var ws;

function BroadcastLine(line) {
	logsTxt.innerHTML = logsTxt.innerHTML + line + "\n";
}

function StartLogs(hostname) {
    if (hostname == null || hostname == "") {
        return;
    }
    logsTxt.innerHTML = '';
    if (ws) {
        ws.close();
    }
	BroadcastLine("starting logs...");
	ws = GetWebsocket('ws://127.0.0.1:8080/logstream/' + hostname);
}
