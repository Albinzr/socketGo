package socket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
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
	IP         string   `json:"ip"`
	Aid        string   `json:"aid"`
	Sid        string   `json:"sid"`
	StartTime  int64    `json:"startTime"`
	EndTime    int64    `json:"endTime"`
	ErrorCount int      `json:"errorCount"`
	ClickCount int      `json:"clickCount"`
	PageCount  int      `json:"pageCount"`
	Initial    bool     `json:"initial"`
	Tags       []string `json:"tags"`
	Urls       []string `json:"urls"`
	Username   string   `json:"username"`
	ID         string   `json:"id"`
	Sex        string   `json:"sex"`
	Age        int      `json:"age"`
	Email      string   `json:"email"`
	InitialURL string   `json:"initialUrl"`
	ExitURL    string   `json:"exitUrl"`
}

var upgrader = websocket.Upgrader{
	HandshakeTimeout: 2 * time.Minute,
}

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
		soc.EndTime = time.Now().Unix() * 1000
		c.OnDisconnect(soc)
		conn.Close()
	}()
	c.readMsg(soc)
}

func getMapValue(obj map[string]interface{}, key string) int {
	if item, found := obj[key]; found {

		if value, ok := item.(float64); ok {

			return int(value)
		} else {
			return 0
		}

	} else {
		return 0
	}
}

func getStats(stats string) map[string]interface{} {

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(stats), &raw); err != nil {
		return nil
	}
	return raw
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
		args := strings.Split(msg, "&%&")
		if len(args) == 0 {
			return
		}
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
		case "/stats", "/update", "/userInfo": // /stats <name> value ack

			if len(args) >= 3 {
				key := args[1]
				value := args[2]

				switch key {
				//stats
				// /stats <name> value ack ---------------------------names:[clickCount,errorCount,pageCount]
				// eg:- /stats clickCount 30 er34
				case "clickCount":
					if clickCount, err := strconv.Atoi(value); err == nil {
						s.ClickCount = clickCount
					}
				case "errorCount":
					if errorCount, err := strconv.Atoi(value); err == nil {
						s.ErrorCount = errorCount
					}
				case "pageCount":
					if pageCount, err := strconv.Atoi(value); err == nil {
						s.PageCount = pageCount
					}
				//update
				// /update <name> value ack ---------------------------names:[tag,URL,initialUrl]
				// eg:- /update initialUrl www.google.com er34
				case "tag":
					s.Tags = append(s.Tags, value)
				case "url":
					s.Urls = append(s.Urls, value)
					s.ExitURL = value
				case "initialUrl":
					s.Urls = append(s.Urls, value)
					s.InitialURL = value
					//userInfo
					// /userInfo <name> value ack ---------------------------names:[username,sex,id,age,email]
					// eg:- /userInfo username albin er34
				case "username":
					s.Username = value
				case "sex":
					s.Sex = value
				case "id":
					s.ID = value
				case "email":
					s.Email = value
				case "age":
					if age, err := strconv.Atoi(value); err == nil {
						s.Age = age
					}
				}

				s.Write("ack " + args[3])
			}

		case "/track":
			if len(args) >= 3 {
				enMsg := args[1]
				c.OnRecive(s, channel, enMsg)
				s.Write("ack " + args[2])
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

				if len(args) >= 4 && args[3] == "initial" {
					s.Initial = true
				} else {
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
			break
		default:

			fmt.Println("***********ERROR*****************")
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
