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
	"sync"
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

type BroadcastMessage struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

func (p BroadcastMessage) jsonDump() string {
	bytes, err := json.Marshal(p)
	if err != nil {
		log.Fatal(err)
	}
	return string(bytes)
}

func pipeProcesser(clientList map[string]*Client) {
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := scanner.Text()
			log.Println("STDIN=[" + line + "]")
			p := BroadcastMessage{time.Now(), line}
			s := p.jsonDump()
			for _, v := range clientList {
				v.Socket.Emit("hoge", s)
			}

		}
		if err := scanner.Err(); err != nil {
			log.Println("Error: could not reading standard input")
		}
	}()
}

type Client struct {
	Socket    socketio.Socket
	Jwt       string
	ParsedJwt jwt.MapClaims
}

func (c *Client) jwtCheck() (jwt.MapClaims, error) {
	hmacSecret := []byte("secret key")
	tokenString := c.Jwt
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return hmacSecret, nil
	})
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		c.ParsedJwt = claims
		return claims, nil
	} else {
		return nil, err
	}
}

func SocketIoServer() *socketio.Server {
	server, err := socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}
	clientList := map[string]*Client{}
	lock := sync.RWMutex{}
	go pipeProcesser(clientList)

	server.On("connection", func(so socketio.Socket) {

		log.Println("[Connected] " + so.Id())
		so.Join("chat")
		lock.Lock()
		clientList[so.Id()] = &Client{so, "", nil}
		lock.Unlock()

		so.On("authRequest", func(msg string) {
			parsedObj := struct {
				Token string
			}{msg}
			if err := json.Unmarshal([]byte(msg), &parsedObj); err != nil {
				log.Println("[authRequest] " + so.Id() + " JsonParseError")
				return
			}
			lock.Lock()
			clientList[so.Id()].Jwt = parsedObj.Token
			_, err := clientList[so.Id()].jwtCheck()
			lock.Unlock()
			if err != nil {
				log.Println("[authRequest] " + so.Id() + " Failed")
			} else {
				log.Println("[authRequest] " + so.Id() + " Successfully")
			}
		})

		so.On("control", func(msg string) {
			log.Println("control " + so.Id() + "[" + msg + "]")
			// log.Println("emit:", so.Emit("chat message", msg))
			// so.BroadcastTo("chat", "chat message", msg)
		})

		so.On("disconnection", func() {
			lock.Lock()
			delete(clientList, so.Id())
			lock.Unlock()
			log.Println("[Disconnected] " + so.Id())
		})
	})
	server.On("error", func(so socketio.Socket, err error) {
		log.Println("error:", err)
	})
	return server
}
