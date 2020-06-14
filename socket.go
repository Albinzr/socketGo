package socket

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"runtime"
	"runtime/debug"
	"strings"
)

// Config -
type Config struct {
	Network      string
	Address      string
	OnConnect    func(conn net.Conn)
	OnDisconnect func(conn net.Conn)
	OnRecive     func(conn net.Conn, channel string, msg string)
}

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
		c.OnConnect(conn)
		go c.client(conn)
	}
}

func (c *Config) client(conn net.Conn) {
	buf := bufio.NewReader(conn)
	//TODO: - if buffer size is to big discard data from buffer
	for {
		msg, err := buf.ReadString('\n')
		fmt.Println("****************************")
		fmt.Println(msg)
		fmt.Println("****************************")
		if err != nil {
			fmt.Println("Socket connection closed for reason:-->", err.Error())
			c.OnDisconnect(conn)
			conn.Close()
			PrintMemUsage()
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
				c.OnRecive(conn, channel, msg)
			}
		case "PROXY":
			fmt.Println(msg)
		default:
			fmt.Println("unknown command:", channel)
			fmt.Println("Connection will now close ----CLOSED----")
			conn.Close()
		}
	}
}

//PrintMemUsage -
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
	fmt.Printf("\tMemory Freed = %v\n", bToMb(m.Frees))

	runtime.GC()
	debug.FreeOSMemory()
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
