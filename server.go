package gomux

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Mux struct {
	Sessions map[int]*Session
}

func NewMux() *Mux {
	return &Mux{Sessions: make(map[int]*Session)}
}

func (m *Mux) Connect(i int) *Session {
	s, ok := m.Sessions[i]
	if ok {
		return s
	}
	s = NewSession()
	m.Sessions[i] = s
	return s
}

func (m *Mux) ListenAndServe() error {
	router := m.router()
	l, e := net.Listen("unix", SERVER)
	if e != nil {
		panic("could not listen to " + SERVER)
	}
	httpServer := http.Server{Addr: SERVER, Handler: router}
	return httpServer.Serve(l)
}

func (m *Mux) postSessionHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	sessionId, err := strconv.Atoi(params["id"])
	if err != nil {
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
	}()

	fmt.Fprintf(stdout, "HTTP/1.1 200 OK\r\nContent-Type: application/vnd.gomux.raw-stream\r\n\r\n")
	// create or get the session attached with session id
	session := m.Connect(sessionId)
	<-session.Attach(stdin, stdout)
}

func (m *Mux) router() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/session/{id:.*}", m.postSessionHandler).Methods("POST")
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
