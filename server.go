package gomux

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gorilla/mux"
)

type Mux struct {
	Sessions       map[string]*Session
	server         string
	events         chan Message
	eventListeners []chan Message
}

func NewMux(server string) *Mux {
	return &Mux{
		Sessions: make(map[string]*Session),
		server:   server,
		events:   make(chan Message, 1),
	}
}

func (m *Mux) Connect(i string) *Session {
	s, ok := m.Sessions[i]
	if ok {
		s.MaybeRecreate()
		return s
	}
	s = NewSession(len(m.Sessions))
	m.Sessions[i] = s
	return s
}

func (m *Mux) Close() {
	for _, s := range m.Sessions {
		s.Close()
	}
}

func (m *Mux) ListenAndServe() error {
	router := m.router()
	l, e := net.Listen("unix", m.server)
	if e != nil {
		panic("could not listen to " + m.server)
	}
	httpServer := http.Server{Addr: m.server, Handler: router}
	// catch signals and cleanup
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		sig := <-c
		Debugf("Received signal '%v', exiting\n", sig)
		os.RemoveAll(m.server)
		m.Close()
		os.Exit(0)
	}()
	// listen and broadcast all events
	go m.listenAndBroadcastEvent()
	Debugln("Starting gomux server at " + m.server)
	return httpServer.Serve(l)
}

func (m *Mux) postResizeHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	sessionId, _ := params["id"]
	id := r.PostFormValue("id")
	width, _ := strconv.Atoi(r.PostFormValue("w"))
	height, _ := strconv.Atoi(r.PostFormValue("h"))
	session := m.Connect(sessionId)
	session.Resize(id, width, height)
	// broadcast this event
	ws := session.Winsize()
	m.events <- Message{
		Event:           "resize",
		SessionId:       sessionId,
		SessionServerId: session.ServerId,
		Width:           int(ws.Width),
		Height:          int(ws.Height),
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m *Mux) postSessionHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	sessionId, ok := params["id"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid session id"))
		return
	}

	stdin, stdout, err := hijackServer(w)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("couldnt create connection"))
		return
	}

	defer func() {
		if tcpc, ok := stdin.(*net.TCPConn); ok {
			tcpc.CloseWrite()
		} else {
			stdin.Close()
		}
	}()

	defer func() {
		if tcpc, ok := stdout.(*net.TCPConn); ok {
			tcpc.CloseWrite()
		} else if closer, ok := stdout.(io.Closer); ok {
			closer.Close()
		}
	}()

	defer func() {
		// do cleanup here for session
		fmt.Println("closed")
	}()

	// create or get the session attached with session id
	session := m.Connect(sessionId)
	fmt.Fprintf(stdout, "HTTP/1.1 200 OK\r\nContent-Type: application/vnd.gomux.raw-stream\r\n\r\n")
	m.events <- Message{
		Event:           "new_client",
		SessionId:       sessionId,
		SessionServerId: session.ServerId,
	}

	<-session.Attach(stdin, stdout)

	// if there isnt any more connections to this session, just broadcast that
	// the session is closed
	if session.NumConn == 0 {
		m.events <- Message{
			Event:           "session_close",
			SessionId:       sessionId,
			SessionServerId: session.ServerId,
		}
		delete(m.Sessions, sessionId)
	}
}

func (m *Mux) getInfoHandler(w http.ResponseWriter, r *http.Request) {
	activeSessions := []string{}
	for sessionId, _ := range m.Sessions {
		activeSessions = append(activeSessions, sessionId)
	}
	info, err := json.Marshal(Info{ActiveSessions: activeSessions})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("failed to get info of gomux server"))
		return
	}
	w.Write(info)
}

// emits events to the listener, this can be implemented by clients to create
// new terminals / resize existing terminals etc
func (m *Mux) getEventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	eventListener := make(chan Message)
	m.eventListeners = append(m.eventListeners, eventListener)
	flusher := w.(http.Flusher)
	// flush the message down http.ResponseWriter
	flush := func(b []byte) error {
		_, err := w.Write(b)
		flusher.Flush()
		return err
	}
	// send the event to http.ResponseWriter
	sendEvent := func(event *Message) error {
		b, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("JSON error")
		}
		flush(b)
		return err
	}
	flush(nil)
	// send all current sessions info as the first message
	sessions := []SessionInfo{}
	for sessionId, session := range m.Sessions {
		sessions = append(sessions, SessionInfo{
			SessionId:       sessionId,
			SessionServerId: session.ServerId,
		})
	}
	err := sendEvent(&Message{Event: "sessions", Sessions: sessions})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// listening to all other events after
	for event := range eventListener {
		err := sendEvent(&event)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (m *Mux) listenAndBroadcastEvent() {
	// multiplex events to multiple listeners
	for event := range m.events {
		for _, w := range m.eventListeners {
			w <- event
		}
	}
}

func (m *Mux) router() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/info", m.getInfoHandler).Methods("GET")
	r.HandleFunc("/events", m.getEventsHandler).Methods("GET")
	r.HandleFunc("/session/{id:.*}", m.postSessionHandler).Methods("POST")
	r.HandleFunc("/session/{id:.*}/resize", m.postResizeHandler).Methods("PUT")
	return r
}

func hijackServer(w http.ResponseWriter) (io.ReadCloser, io.Writer, error) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return nil, nil, err
	}
	// Flush the options to make sure the client sets the raw mode
	conn.Write([]byte{})
	return conn, conn, nil
}
