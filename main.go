package main

import (
	"bufio"
	"encoding/json"
	"github.com/googollee/go-socket.io"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"net/http"
	"os"
	"time"
)

type customServer struct {
	Server *socketio.Server
}

func (s *customServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	origin := r.Header.Get("Origin")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	s.Server.ServeHTTP(w, r)
}

func main() {

	if terminal.IsTerminal(0) {
		log.Fatal("no pipe")
	}

	ioServer := SocketIoServer()
	wsServer := new(customServer)
	wsServer.Server = ioServer
	http.Handle("/socket.io/", wsServer)
	port := "3000"
	log.Println("[Main] Starting Server Port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

type Packet struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

func (p Packet) json() string {
	bytes, err := json.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}
	return string(bytes)
}

func pipeProcesser(soList map[string]socketio.Socket) {

	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			p := Packet{time.Now(), line}
			s := p.json()
			for _, so := range soList {
				so.Emit("hoge", s)
			}

		}
		if err := scanner.Err(); err != nil {
			log.Println("Error: could not reading standard input")
		}
		log.Println("Owari")
	}()
}

func SocketIoServer() *socketio.Server {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	soList := map[string]socketio.Socket{}
	go pipeProcesser(soList)

	server.On("connection", func(so socketio.Socket) {

		log.Println("[Connected] " + so.Id())

		soList[so.Id()] = so
		so.Join("chat")

		so.On("msg", func(msg string) {
			log.Println("RECV [" + msg + "]")
			// log.Println("emit:", so.Emit("chat message", msg))
			// so.BroadcastTo("chat", "chat message", msg)
		})

		so.On("disconnection", func() {
			delete(soList, so.Id())
			log.Println("[Disconnected] " + so.Id())
		})

	})
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})
	return server
}
