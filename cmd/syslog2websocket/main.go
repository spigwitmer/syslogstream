package main

import (
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"gopkg.in/mcuadros/go-syslog.v2"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

const (
	// pulled straight out of gorilla websocket examples

	// Time allowed to write the file to the client.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the client.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Poll file for changes with this period.
	filePeriod = 10 * time.Second
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     CheckOrigin,
	}
	TaskIDRegex     = regexp.MustCompile(`/logstream/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)
	ClientsByTaskID map[string]map[*websocket.Conn]chan string
)

func CheckOrigin(r *http.Request) bool {
	return true
}

func init() {
	ClientsByTaskID = make(map[string]map[*websocket.Conn]chan string)
}

func RegisterClient(taskID string, ws *websocket.Conn) chan string {
	if ClientsByTaskID[taskID] == nil {
		ClientsByTaskID[taskID] = make(map[*websocket.Conn]chan string)
	}
	ClientsByTaskID[taskID][ws] = make(chan string)
	return ClientsByTaskID[taskID][ws]
}

func UnregisterClient(taskID string, ws *websocket.Conn) {
	close(ClientsByTaskID[taskID][ws])
	delete(ClientsByTaskID[taskID], ws)
}

func BroadcastLogMessage(taskID string, msg string) {
	if _, ok := ClientsByTaskID[taskID]; ok {
		for _, broadcastChan := range ClientsByTaskID[taskID] {
			broadcastChan <- msg
		}
	}
}

func writer(ws *websocket.Conn, lastMod time.Time, taskID string) {
	defer ws.Close()
	broadcastChan := RegisterClient(taskID, ws)
	defer UnregisterClient(taskID, ws)

	for {
		select {
		case msg := <-broadcastChan:
			if err := ws.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				log.Error(err)
				break
			}
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request, taskID string) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error(err)
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	var lastMod time.Time
	if n, err := strconv.ParseInt(r.FormValue("lastMod"), 16, 64); err == nil {
		lastMod = time.Unix(0, n)
	}

	log.Info("starting writer")
	writer(ws, lastMod, taskID)
}

func logsRouter(w http.ResponseWriter, r *http.Request) {
	parts := TaskIDRegex.FindStringSubmatch(r.URL.Path)
	if parts[1] != "" {
		log.Infof("upgrading connection: %s %s", r.Method, r.URL.Path)
		serveWs(w, r, parts[1])
		return
	}
	w.WriteHeader(404)
}

func main() {
	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.RFC5424)
	server.SetHandler(handler)
	if err := server.ListenUDP("127.0.0.1:514"); err != nil {
		log.Panic(err)
	}
	if err := server.Boot(); err != nil {
		log.Panic(err)
	}

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			if hostname, ok := logParts["hostname"]; ok && hostname != nil && hostname != "" {
				BroadcastLogMessage(fmt.Sprintf("%s", hostname), fmt.Sprintf("%s", logParts["message"]))
			}
		}
	}(channel)

	log.Info("Starting syslog server on :514")
	defer server.Kill()
	go server.Wait()

	http.HandleFunc("/", logsRouter)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
