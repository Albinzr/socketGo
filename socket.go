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

	fmt.Println("server started on", c.Address)

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

		// soc = clientMap[conn.RemoteAddr().String()]

		c.OnConnect(soc)
		go c.client(soc)
	}
}

func (c *Config) client(s Socket) {
	buf := bufio.NewReader(s.conn)

	//TODO: - if buffer size is to big discard data from buffer
	for {
		msg, err := buf.ReadString('\n')
		fmt.Println("****************************")
		fmt.Println(msg)
		fmt.Println("****************************")
		if err != nil {
			fmt.Println("Socket connection closed for reason:-->", err.Error())
			c.OnDisconnect(s)
			// delete(clientMap, s.conn.RemoteAddr().String())
			s.conn.Close()
			return
		}
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
			fmt.Println(msg)
			if len(args) >= 2 {
				s.IP = args[2]
				s.StartTime = time.Now().UnixNano() / int64(time.Millisecond)
			}
		default:
			fmt.Println("unknown command:", channel)
			fmt.Println("Connection will now close ----CLOSED----")
			c.OnDisconnect(s)
			// delete(clientMap, s.conn.RemoteAddr().String())
			s.conn.Close()
		}
	}
}

func (s *Socket) Write(msg string) {
	s.conn.Write([]byte(msg))
}
