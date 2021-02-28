package main

import (
	"container/list"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"gopkg.in/mcuadros/go-syslog.v2"
	"net/http"
	"regexp"
	"strconv"
	"strings"
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
	LogstreamRegex    = regexp.MustCompile(`^/logstream/([-0-9A-Za-z.]+)$`)
	HostnameRegex     = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*$`)
	ClientsByHostname map[string]map[*websocket.Conn]chan string
	BuffersByHostname map[string]*list.List

	bufferMaxLines = flag.Int("buffer-max-lines", 1024, "Max number of lines to buffer for each hostname")
)

func CheckOrigin(r *http.Request) bool {
	return true
}

func init() {
	ClientsByHostname = make(map[string]map[*websocket.Conn]chan string)
	BuffersByHostname = make(map[string]*list.List)
	log.SetLevel(log.DebugLevel)
}

func RegisterClient(syslogHostname string, ws *websocket.Conn) chan string {
	if ClientsByHostname[syslogHostname] == nil {
		ClientsByHostname[syslogHostname] = make(map[*websocket.Conn]chan string)
	}
	ClientsByHostname[syslogHostname][ws] = make(chan string)
	return ClientsByHostname[syslogHostname][ws]
}

func UnregisterClient(syslogHostname string, ws *websocket.Conn) {
	close(ClientsByHostname[syslogHostname][ws])
	delete(ClientsByHostname[syslogHostname], ws)
}

func BroadcastLogMessage(syslogHostname string, msg string) {
	if _, ok := ClientsByHostname[syslogHostname]; ok {
		for _, broadcastChan := range ClientsByHostname[syslogHostname] {
			broadcastChan <- msg
		}
	}
}

func AddToBuffer(syslogHostname string, msg string) {
	var msgList *list.List
	if BuffersByHostname[syslogHostname] == nil {
		BuffersByHostname[syslogHostname] = list.New()
	}
	msgList = BuffersByHostname[syslogHostname]
	msgList.PushBack(msg)
	if msgList.Len() > *bufferMaxLines {
		msgList.Remove(msgList.Front())
	}
}

func writer(ws *websocket.Conn, lastMod time.Time, syslogHostname string) {
	defer ws.Close()
	broadcastChan := RegisterClient(syslogHostname, ws)
	defer UnregisterClient(syslogHostname, ws)

	// write current buffer
	if BuffersByHostname[syslogHostname] != nil {
		var priorLogs string
		for priorMsg := BuffersByHostname[syslogHostname].Front(); priorMsg != nil; priorMsg = priorMsg.Next() {
			if priorLogs != "" {
				priorLogs += "\n" + priorMsg.Value.(string)
			} else {
				priorLogs += priorMsg.Value.(string)
			}
		}

		if err := ws.WriteMessage(websocket.TextMessage, []byte(priorLogs)); err != nil {
			log.Error(err)
		}
	}

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

func serveWs(w http.ResponseWriter, r *http.Request, syslogHostname string) {
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

	writer(ws, lastMod, syslogHostname)
}

func validFQDN(fqdn string) bool {
	parts := strings.Split(fqdn, ".")
	for _, part := range parts {
		if !HostnameRegex.Match([]byte(part)) {
			return false
		}
	}
	return true
}

func logsRouter(w http.ResponseWriter, r *http.Request) {
	parts := LogstreamRegex.FindStringSubmatch(r.URL.Path)
	log.Debugf("%+v", parts)
	if len(parts) < 2 {
		log.Debugf("%s: len(parts) < 2", r.URL.Path)
		w.WriteHeader(404)
		return
	}
	if !validFQDN(parts[1]) {
		log.Debugf("%s: Not a valid FQDN", r.URL.Path)
		w.WriteHeader(400)
		return
	}
	if parts[1] != "" {
		log.Infof("upgrading connection: %s %s", r.Method, r.URL.Path)
		serveWs(w, r, parts[1])
		return
	}
}

func main() {
	flag.Parse()

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
				hostnameStr := fmt.Sprintf("%s", hostname)
				msgStr := fmt.Sprintf("%s", logParts["message"])
				AddToBuffer(hostnameStr, msgStr)
				BroadcastLogMessage(hostnameStr, msgStr)
			}
		}
	}(channel)

	log.Info("Starting syslog server on :514")
	defer server.Kill()
	go server.Wait()

	http.HandleFunc("/", logsRouter)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
