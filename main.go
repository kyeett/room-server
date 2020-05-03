package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"gopkg.in/olahol/melody.v1"
)


type Server struct {
	router   http.Handler
	m        *melody.Melody
	lock     *sync.Mutex
	sessions map[string]*melody.Session
}

const (
	keyRoomID   = "room_id"
	keyClientID = "client_id"
)

func main() {
	s := New()
	port, _ := os.LookupEnv("PORT")
	if port == "" {
		port = "5000"
	}
	if err := s.Start(port); err != nil {
		log.Fatal(err)
	}
}

func New() *Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)

	m := melody.New()
	m.Config.MaxMessageSize = 2048
	r.Get("/room/{room_id}", func(w http.ResponseWriter, r *http.Request) {
		roomID := chi.URLParam(r, "room_id")
		m.HandleRequestWithKeys(w, r, map[string]interface{}{
			keyRoomID: roomID,
		})
	})

	s := &Server{
		router:   r,
		m:        m,
		lock:     new(sync.Mutex),
		sessions: map[string]*melody.Session{},
	}

	m.HandleConnect(s.handleConnect)
	m.HandleMessage(s.handleMessage)

	return s
}

func (s *Server) handleConnect(_ *melody.Session) {
	fmt.Println("successfully connected")
}

func (s *Server) handleMessage(sess *melody.Session, msg []byte) {
	fmt.Println("broadcasting message")
	filterToRoom := func(other *melody.Session) bool {
		return other != sess && isSameRoom(sess, other)
	}
	if err := s.m.BroadcastFilter(msg, filterToRoom); err != nil {
		fmt.Printf("Error broadcasting connection message: %s\n", err)
		return
	}
}

func isSameRoom(self, other *melody.Session) bool {
	selfRoom, _ := self.Get(keyRoomID)
	otherRoom, _ := other.Get(keyRoomID)
	return selfRoom == otherRoom
}

func (s *Server) Start(port string) error {
	fmt.Println("starting")
	return http.ListenAndServe(":" + port, s.router)
}
