package socket

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
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
	Status    string `json:"type"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
	conn      net.Conn
}

// var clientMap map[string]Socket

//Init -
func (c *Config) Init() {
	ln, err := net.Listen(c.Network, c.Address)
	if err != nil {
		log.Fatal("Cannot start net.Listen", err)
		return
	}
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error form socker ln.accept()", err)
			return
		}
		fmt.Println("connected", conn.RemoteAddr())

		var soc = Socket{
			conn: conn,
		}

		c.OnConnect(soc)
		go c.client(soc)
	}
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
				msg := args[1]
				c.OnRecive(s, channel, msg)
			}
		case "PROXY":
			if len(args) >= 2 {
				s.IP = args[2]
				s.StartTime = time.Now().UnixNano() / int64(time.Millisecond)
			}
		case "/info":
			if len(args) >= 2 {
				s.Sid = args[1]
				s.Aid = args[2]
			}
		default:
			fmt.Println("unknown command:", channel)
			fmt.Println("****************************")
			fmt.Println(msg)
			fmt.Println("****************************")
			fmt.Println("Connection will now close ----CLOSED----")
			//
			c.internalClose(s)
		}
	}
}

//Write - write back to connection
func (s *Socket) Write(msg string) {
	s.conn.Write([]byte(msg))
}

//Close - close connection
func (s *Socket) Close(msg string) {
	s.conn.Close()
}

func (c *Config) internalClose(s Socket) {
	s.conn.Close()
	c.OnDisconnect(s)
}
