package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("Error connecting:", err)
		os.Exit(1)
	}
	defer conn.Close()

	reader := bufio.NewReader(os.Stdin)

	go func() {
		serverReader := bufio.NewReader(conn)
		for {
			message, err := readMessage(serverReader)
			if err != nil {
				fmt.Println("Server Disconnected.")
				return
			}
			fmt.Println(message)
		}
	}()

	for {
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		sendMessage(conn, text)
	}
}

func sendMessage(conn net.Conn, message string) {
	msgBytes := []byte(message)
	length := uint16(len(msgBytes))
	lengthByte := make([]byte, 2)
	binary.BigEndian.PutUint16(lengthByte, length)
	conn.Write(lengthByte)
	conn.Write(msgBytes)
}

func readMessage(reader *bufio.Reader) (string, error) {
	lengthBytes := make([]byte, 2)
	_, err := reader.Read(lengthBytes)
	if err != nil {
		return "", err
	}

	length := binary.BigEndian.Uint16(lengthBytes)
	msgBytes := make([]byte, length)
	_, err = reader.Read(msgBytes)
	if err != nil {
		return " ", err
	}
	return string(msgBytes), nil
}
