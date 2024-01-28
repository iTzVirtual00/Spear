package main

import (
	"crypto/tls"
	"fmt"
	"github.com/akamensky/argparse"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// receive a file from a future connection from a sender
func listenReceiver(port int, file string) {
	WsNewListener(port, func(c *websocket.Conn) {
		defer c.Close()
		receiver(c, file)
		os.Exit(0)
	})
}

// receive a file from a remote listener
func sendReceiver(addr string, file string) {
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	c, _, err := dialer.Dial(fmt.Sprintf("wss://%s", addr), nil)
	if err != nil {
		log.Println("Cannot init connection:", err)
		return
	}
	defer c.Close()
	receiver(c, file)
	os.Exit(0)
}

// send a file to a future connection from a receiver
func listenSender(port int, file string) {
	WsNewListener(port, func(c *websocket.Conn) {
		defer c.Close()
		sender(c, file)
		os.Exit(0)
	})
}

// send a file to a remote listener
func sendSender(addr string, file string) {
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	c, _, err := dialer.Dial(fmt.Sprintf("wss://%s", addr), nil)
	if err != nil {
		log.Println("Cannot init connection:", err)
		return
	}
	defer c.Close()
	sender(c, file)
	os.Exit(0)
}

func sender(c *websocket.Conn, file string) {
	defer c.Close()
	fileInfo, err := os.Stat(file)
	if err != nil {
		fmt.Println("Cannot stat file:", err)
		return
	}

	err = c.WriteMessage(websocket.BinaryMessage, []byte(strconv.FormatInt(fileInfo.Size(), 10)))
	if err != nil {
		fmt.Println("IO error:", err)
		return
	}
	hFile, err := os.Open(file)
	if err != nil {
		fmt.Println("Open file err:", err)
		return
	}
	buf := make([]byte, 1024)
	for {
		bytesRead, err := hFile.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Done sending")
				return
			}
			fmt.Println("Cannot bytesRead: ", err)
			return
		}
		if bytesRead == 0 {
			break
		}
		if bytesRead < 1024 {
			buf = buf[:bytesRead]
		}
		err = c.WriteMessage(websocket.BinaryMessage, buf)
		if err != nil {
			fmt.Println("IO error:", err)
			return
		}
	}
}

func receiver(c *websocket.Conn, file string) {
	_, message, err := c.ReadMessage()
	if err != nil {
		log.Println("read error:", err)
		return
	}
	size, err := strconv.Atoi(strings.TrimRight(string(message), "\r\n"))
	if err != nil {
		log.Println("parse size error:", err)
		return
	}
	hFile, err := os.Create(file)
	fmt.Printf("writing to %s(%d)\n", file, size)
	if err != nil {
		log.Println("parse size error:", err)
		return
	}
	for size > 0 {
		_, data, err := c.ReadMessage()
		if err != nil {
			log.Println("read error:", err)
			return
		}
		written, err := hFile.Write(data)
		if err != nil {
			log.Println("write error:", err)
			return
		}
		size -= written
	}
	fmt.Printf("Done writing")
	os.Exit(0)
}

func main() {
	listenOptions := &argparse.Options{Required: false, Default: -1, Help: "Activate listen mode"}
	addrOptions := &argparse.Options{Required: false, Default: "", Help: "Address"}
	fileOptions := &argparse.Options{Required: true, Help: "File"}

	// Create new parser object
	parser := argparse.NewParser("spear", "Standalone utility to send and receive file with ease")
	// Create string flag

	receiveCmd := parser.NewCommand("receive", "receive a file")
	// receiveCmd.Int("l", "listen", listenOptions)

	sendCmd := parser.NewCommand("send", "send a file")
	// sendCmd.Int("l", "listen", listenOptions)

	listen := parser.Int("l", "listen", listenOptions)
	addr := parser.String("a", "addr", addrOptions)
	file := parser.String("f", "file", fileOptions)

	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	if (*addr == "") != (*listen != -1) { // xor
		fmt.Print(parser.Usage(err))
		os.Exit(1)
	}

	if receiveCmd.Happened() {
		if *addr != "" {
			sendReceiver(*addr, *file)
		} else {
			listenReceiver(*listen, *file)
		}
	} else if sendCmd.Happened() {
		if *addr != "" {
			sendSender(*addr, *file)
		} else {
			listenSender(*listen, *file)
		}
	}
}
