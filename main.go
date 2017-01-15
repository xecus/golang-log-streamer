package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
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

type BroadcastMessageModel struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

func (p BroadcastMessageModel) jsonDump() string {
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
			p := BroadcastMessageModel{time.Now(), line}
			s := p.jsonDump()
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

type AuthRequestModel struct {
	Token string `json:"token"`
}

func parseAuthRequestModel(s string) AuthRequestModel {
	var a AuthRequestModel
	err := json.Unmarshal([]byte(s), &a)
	if err != nil {
	}
	return a
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

		so.On("authRequest", func(msg string) {
			authRequest := parseAuthRequestModel(msg)
			tokenString := authRequest.Token
			hmacSampleSecret := []byte("secret key")
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
				}
				return hmacSampleSecret, nil
			})
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				log.Println("[authRequest] Successflly")
				fmt.Println(claims)
			} else {
				log.Println("[authRequest] Unauthorized")
				fmt.Println(err)
			}
		})

		so.On("control", func(msg string) {
			log.Println("control [" + msg + "]")
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
