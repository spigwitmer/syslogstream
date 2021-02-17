package main

import (
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"gopkg.in/mcuadros/go-syslog.v2"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

const (
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
	}
	TaskIDRegex = regexp.MustCompile(`/logstream/([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)
)

func reader(ws *websocket.Conn) {
	defer ws.Close()
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { ws.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			break
		}
	}
}

func writer(ws *websocket.Conn, lastMod time.Time, taskID string) {
	defer ws.Close()

	log.Infof("About to write something")
	c := time.Tick(5 * time.Second)
	// write what's currently in buffer for
	for _ = range c {
		log.Infof("Writing something")
		ws.WriteMessage(websocket.TextMessage, []byte("hello"))
	}
}

func serveWs(w http.ResponseWriter, r *http.Request, taskID string) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); !ok {
			log.Println(err)
		}
		return
	}

	var lastMod time.Time
	if n, err := strconv.ParseInt(r.FormValue("lastMod"), 16, 64); err == nil {
		lastMod = time.Unix(0, n)
	}

	go writer(ws, lastMod, taskID)
	reader(ws)
}

func logsRouter(w http.ResponseWriter, r *http.Request) {
	if parts := TaskIDRegex.FindStringSubmatch(r.URL.Path); parts[1] != "" {
		log.Infof("%s %s [upgrading]", r.Method, r.URL.Path)
		serveWs(w, r, parts[1])
		return
	}
	log.Infof("%s %s [404]", r.Method, r.URL.Path)
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
			if msg, ok := logParts["message"]; ok {
				log.Infof("%s", msg)
			}
		}
	}(channel)

	log.Info("Starting syslog server on :514")
	defer server.Kill()
	go server.Wait()

	http.HandleFunc("/", logsRouter)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
