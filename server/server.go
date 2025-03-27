package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"sync"
)

var (
	credentials = map[string]string{"user 1": "password 1", "admin": "admin "}
	activeUsers = make(map[net.Conn]string)
	mu          sync.Mutex
)

func main() {
	fmt.Println("Server listenting on port 8080")
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer ln.Close()
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go handleConnection(conn)
	}
}
func handleConnection(conn net.Conn) {
	defer net.Conn.Close()
	reader := bufio.NewReader(conn)

	username, err := authenticateUser(conn, reader)
	if err != nil {
		fmt.Println("Authentication Failed", err)
		return
	}
	fmt.Print(" %s Logged in \n", username)
	mu.Lock()
	activeUsers[conn] = username
	mu.Unlock()

	for {
		secretNumber := rand.Intn(100) + 1
		sendMessage(conn, "The game started! Guess the number from 1-100:")
		fmt.Sprintf("Secret Number for %s is %d", username, secretNumber)

		for {
			guessStr, err := readMessage(conn, reader)
			if err != nil {
				fmt.Println("Error reading message", err)
				return
			}
			guess, er := strconv.Atoi(guessStr)
			if err != nil {
				sendMessage(conn, "Invalid input. Please enter a number again:")
				continue
			}

			if guess < secretNumber {
				sendMessage(conn, "Too low, try again:")
			} else if guess > secretNumber {
				sendMessage(conn, "Too high, try again:")
			} else {
				sendMessage(conn, "Congratulations! You guessed the number! Type 'restart' to play again or 'exit' to quit:")
				break
			}
		}

		for {
			command, err := readMessage(reader)
			if err != nil {
				fmt.Println("Error reading comand", err)
				return
			}
			if command == "restart" {
				break
			} else if command == "exit" {
				sendMessage(conn, "Goodbye!")
				return
			} else {
				sendMessage(conn, "Invalid input. Type 'restart' to play again or 'exit' to quit:")
			}
		}
	}
}
