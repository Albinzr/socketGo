package socket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// Config -
type Config struct {
	Network      string
	Address      string
	OnConnect    func(s *Socket)
	OnDisconnect func(s *Socket)
	OnRecive     func(s *Socket, channel string, msg string)
}

//Socket socket data passed for callback
type Socket struct {
	conn       *websocket.Conn
	IP         string `json:"ip"`
	Aid        string `json:"aid"`
	Sid        string `json:"sid"`
	StartTime  int64  `json:"startTime"`
	EndTime    int64  `json:"endTime"`
	ErrorCount int    `json:"errorCount"`
	ClickCount int    `json:"clickCount"`
	PageCount  int    `json:"pageCount"`
	Initial  bool    `json:"initial"`
}

var upgrader = websocket.Upgrader{
	HandshakeTimeout: 2 * time.Minute,
}

var stats map[string]string = make(map[string]string)

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
	var soc = &Socket{
		conn: conn,
		IP:   r.Header.Get("X-Real-IP"),
	}
	defer func() {
		fmt.Println("Connection closed")
		sid := soc.Sid
		soc.EndTime = time.Now().Unix() * 1000

		currentStats := getStats(stats[sid])

		fmt.Println("***********************************")
		fmt.Println(currentStats)
		fmt.Println("***********************************")
		soc.ClickCount = getMapValue(currentStats, "clickCount")
		soc.ErrorCount = getMapValue(currentStats, "errorCount")
		soc.PageCount = getMapValue(currentStats, "pageCount")
		fmt.Println("***********************************")
		fmt.Println(soc.ClickCount)
		fmt.Println(soc.ErrorCount)
		fmt.Println(soc.PageCount)
		fmt.Println("***********************************")
		delete(stats, sid)

		c.OnDisconnect(soc)
		conn.Close()
	}()
	c.readMsg(soc)
}

func getMapValue(obj map[string]interface{}, key string) int {
	if item, found := obj[key]; found {
		if value, ok := item.(int); !ok {
			return value
		} else {
			return 0
		}
	} else {
		return 0
	}
}

func getStats(stats string) map[string]interface{} {
	var result map[string]interface{}
	json.Unmarshal([]byte(stats), &result)
	return result
}

func (c *Config) readMsg(s *Socket) {
	//TODO: - if buffer size is to big discard data from buffer
	for {
		mt, msgBytes, err := s.conn.ReadMessage()
		if err != nil {
			log.Println("msgType: ", mt, "read---------------->:", err)
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
				c.OnRecive(s, channel, enMsg)
				s.Write("ack " + args[2])
			} else {
				//remove write
				s.Write("beacon format wrong (connection will close now) ")
				s.conn.Close()
			}

		case "PROXY":
			if len(args) >= 3 {
				s.IP = args[2]
				s.StartTime = time.Now().Unix() * 1000
			}
		case "/stats": // /stats sid {clickCount:10,errorCount:20,pageCount:4} (json string - no space in json)
			if len(args) >= 3 {
				sid := args[1]
				statData := args[2]
				stats[sid] = statData
				fmt.Println("*************in******",statData)
				fmt.Println("*************Dict******",stats[sid])
			}

		case "/connect":
			if len(s.Sid) > 0 {
				//remove write
				s.Write("Multiple connection (connection will close now) ")
				s.conn.Close()
			}
			if len(args) >= 3 {
				s.Sid = args[1]
				s.Aid = args[2]

				if len(args) >= 4 && args[3] == "initial"{
					s.Initial = true
				}else{
					s.Initial = false
				}
				
				s.StartTime = time.Now().Unix() * 1000
				s.Write("Accepted")
				s.Write(s.Sid + "-" + s.Aid)
				c.OnConnect(s)
			} else {
				s.Write("connect format wrong (connection will close now) ")
				s.conn.Close()
			}


		case "/hb":
			if len(args) == 1 {
				fmt.Println("hb--->")
			}

		default:

			fmt.Println("****************************")
			fmt.Println("unknown command:", channel)
			fmt.Println(msg)
			fmt.Println("Connection will now close ----CLOSED----")
			fmt.Println("****************************")
			s.conn.Close()
		}
	}
}

//Write - write back to connection
func (s *Socket) Write(msg string) {
	err := s.conn.WriteMessage(1, []byte(msg))
	if err != nil {
		fmt.Println("error writing : -> ", msg, "reason ->", err)
	}
}

//Close - close connection
func (s *Socket) Close() {
	err := s.conn.Close()
	if err != nil {
		fmt.Println("Manuel close failled : reason ->", err)
	}

}

func keepAlive(c *websocket.Conn, timeout time.Duration) {
	lastResponse := time.Now()
	c.SetPongHandler(func(msg string) error {
		lastResponse = time.Now()
		return nil
	})

	go func() {
		for {
			err := c.WriteMessage(websocket.PingMessage, []byte("keepalive"))
			if err != nil {
				return
			}
			time.Sleep(timeout / 2)
			if time.Now().Sub(lastResponse) > timeout {
				c.Close()
				return
			}
		}
	}()
}
