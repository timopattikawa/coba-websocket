package main

import (
	"flag"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

var addr = flag.String("addr", "localhost:8080", "http service address")
var upgrader = websocket.Upgrader{}

func receiver(w *websocket.Conn, done chan struct{}) {
	defer func(w *websocket.Conn) {
		err := w.Close()
		if err != nil {
			log.Println("Error close connection in receive", err)
		}
	}(w)
	defer close(done)

	w.SetPongHandler(func(appData string) error {
		log.Println("Receive Pong From client: Client still alive")
		return nil
	})

	for {
		_, message, err := w.ReadMessage()
		if err != nil {
			log.Printf("read: %s", err.Error())
			return
		}
		log.Printf("Receive: %s", message)
	}
}

func sender(w *websocket.Conn, done chan struct{}) {
	defer func() {
		err := w.Close()
		if err != nil {
			log.Println(err)
			return
		}
	}()

	messageTicker := time.NewTicker(1 * time.Second)
	defer messageTicker.Stop()

	messagePing := time.NewTicker(5 * time.Second)
	defer messagePing.Stop()

	ctr := 0
breakLoop:
	for {
		select {
		case <-messageTicker.C:
			textData := "hello world"
			if err := w.WriteMessage(websocket.TextMessage, []byte(textData)); err != nil {
				log.Println(err)
				return
			}
			if ctr > 20 {
				break breakLoop
			}
			ctr++
		case <-messagePing.C:
			err := w.WriteControl(websocket.PingMessage, []byte("ping message"), time.Time{})
			if err != nil {
				log.Println(err)
				return
			}
		case <-done:
			return
		}
	}

	err := w.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("Error Close message ", err)
		return
	}
	<-done
}

func echo(rw http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(rw, r, nil)
	if err != nil {
		log.Println("Error upgrade")
		return
	}
	defer func(conn *websocket.Conn) {
		err := conn.Close()
		if err != nil {
			log.Println(err)
			return
		}
	}(conn)

	done := make(chan struct{})

	go receiver(conn, done)
	go sender(conn, done)

	<-done
	log.Println("websocket handler is done")
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/echo", echo)
	//http.HandleFunc("/", home)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
