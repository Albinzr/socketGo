package socket

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	lz "github.com/Albinzr/lzGo"
	ws "github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
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
	conn      net.Conn
}

// var clientMap map[string]Socket

//Init -
func (c *Config) Init() {

	fmt.Println("localhost:8080")
	http.ListenAndServe("localhost:8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			log.Fatal("Cannot start upgrade-------------", err)
		}

		var soc = Socket{
			conn: conn,
		}

		go c.client(soc)

}

func (c *Config) client(s Socket) {
	buf := bufio.NewReader(s.conn)

	//TODO: - if buffer size is to big discard data from buffer
	for {
		msg, err := buf.ReadString('\n')
		if err != nil {
			fmt.Println("Socket connection closed for reason:-->", err.Error())
			s.EndTime = time.Now().UnixNano() / int64(time.Millisecond)
			c.OnDisconnect(s)
			return
		}
		//
		msg = strings.Trim(msg, "\r\n")
		args := strings.Split(msg, " ")
		channel := strings.TrimSpace(args[0])

		switch channel {
		//TODO: - check sum logic, decompression logic
		case "/beacon":
			if len(args) > 1 {
				enMsg := args[1]
				deMsg, err := lz.DecompressFromBase64(enMsg)

				if err != nil {
					c.OnRecive(s, channel, deMsg)
					return
				}

				fmt.Println("error decompressing msg", err)
				return

			}
		case "PROXY":
			if len(args) >= 2 {
				s.IP = args[2]
				s.StartTime = time.Now().UnixNano() / int64(time.Millisecond)
			}
		case "/connect":
			if len(args) >= 2 {
				s.Sid = args[1]
				s.Aid = args[2]
			}
			c.OnConnect(s)
		default:

			fmt.Println("****************************")
			fmt.Println("unknown command:", channel)
			fmt.Println("Connection will now close ----CLOSED----")
			fmt.Println(msg)
			fmt.Println("****************************")

			//
			c.OnDisconnect(s)
		}
	}
}

//Write - write back to connection
func (s *Socket) Write(msg string) {
	s.conn.Write([]byte(msg))
}

//Close - close connection
func (s *Socket) Close() {
	s.conn.Close()
}
