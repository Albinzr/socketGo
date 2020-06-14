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
		if err != nil {
			if err.Error() == "EOF" { //|| strings.Contains(err.Error(), "use of closed network connection") {
				fmt.Println("socket connection closed ********EOF********* ", err.Error())
				c.OnDisconnect(conn)
				conn.Close()
				PrintMemUsage()
				return
			}
			fmt.Println("socket msg reading error", err.Error())
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
		default:
			fmt.Println("unknown command:", channel)
			fmt.Println("Connection will now close ******************************************")
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
