function GetWebsocket(url) {
	let socket = new WebSocket(url);

	socket.onmessage = function(event) {
	  BroadcastLine(event.data);
	};

    /*
	socket.onerror = function(error) {
	  alert(`[error] ${error.message}`);
	};
    */

	return socket;
}

var logsTxt = document.getElementById('logs');

function BroadcastLine(line) {
	logsTxt.innerHTML = logsTxt.innerHTML + line + "\n";
}

function StartLogs() {
	BroadcastLine("starting logs...");
	ws = GetWebsocket('ws://127.0.0.1:8080/logstream/task-36b1308f-e0e2-4d4a-ae98-284f51f39a8a');
}
