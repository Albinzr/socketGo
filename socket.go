package socket

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	lz "github.com/Albinzr/lzGo"
	"github.com/gorilla/websocket"
)

// Config -
type Config struct {
	Network      string
	Address      string
	OnConnect    func(s Socket)
	OnDisconnect func(s Socket)
	OnRecive     func(s Socket, channel string, msg string)
}

//Socket socket data passed for callback
type Socket struct {
	IP        string `json:"ip"`
	Aid       string `json:"aid"`
	Sid       string `json:"sid"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
	conn      *websocket.Conn
}

var upgrader = websocket.Upgrader{}

//Init -
func (c *Config) Init() {
	http.HandleFunc("/", c.processData)
	log.Fatal(http.ListenAndServe(c.Address, nil))
}

func (c *Config) processData(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	var soc = Socket{
		conn: conn,
	}
	defer func() {
		fmt.Println("Connection closed")
		c.OnDisconnect(soc)
		conn.Close()
	}()
	c.readMsg(soc)
}

func (c *Config) readMsg(s Socket) {
	//TODO: - if buffer size is to big discard data from buffer
	for {
		_, msgBytes, err := s.conn.ReadMessage()
		if err != nil {
			log.Println("read---------------->:", err)
			s.conn.Close()
			break
		}

		msg := string(msgBytes)
		msg = strings.Trim(msg, "\r\n")
		args := strings.Split(msg, " ")
		channel := strings.TrimSpace(args[0])

		switch channel {
		//TODO: - check sum logic, decompression logic
		case "/beacon":

			if len(args) >= 2 && len(s.Sid) > 0 {
				enMsg := args[1]
				deMsg, err := lz.DecompressFromBase64(enMsg)
				if err != nil || enMsg == "" {
					s.conn.Close()
					c.OnDisconnect(s)
					fmt.Println("decomperssion failed")
				}
				c.OnRecive(s, channel, deMsg)
				s.Write(args[2])
			} else {
				s.conn.Close()
			}

		case "PROXY":
			if len(args) >= 3 {
				s.IP = args[2]
				s.StartTime = time.Now().UnixNano() / int64(time.Millisecond)
			}
		case "/connect":
			if len(args) >= 3 {
				s.Sid = args[1]
				s.Aid = args[2]
			}
			c.OnConnect(s)
			s.Write("Accepted")
		default:

			fmt.Println("****************************")
			fmt.Println("unknown command:", channel)
			fmt.Println(msg)
			fmt.Println("Connection will now close ----CLOSED----")
			fmt.Println("****************************")
			s.conn.Close()
			c.OnDisconnect(s)
		}
	}
}

//Write - write back to connection
func (s *Socket) Write(msg string) {
	err := s.conn.WriteMessage(1, []byte(msg))
	fmt.Println("error writing : -> ", msg, "reason ->", err)
}

//Close - close connection
func (s *Socket) Close() {
	s.conn.Close()
}
