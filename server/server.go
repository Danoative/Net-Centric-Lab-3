package main

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
)

const userFile = "users.json"

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var (
	users       = make(map[string]string)
	activeUsers = make(map[net.Conn]string)
	mu          sync.Mutex
)

func main() {

	loadUsers()

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
	defer conn.Close()
	reader := bufio.NewReader(conn)

	username, err := authenticateUser(conn, reader)
	if err != nil {
		fmt.Println("Authentication Failed", err)
		return
	}
	fmt.Printf(" %s Logged in \n", username)
	mu.Lock()
	activeUsers[conn] = username
	mu.Unlock()

	for {
		secretNumber := rand.Intn(100) + 1
		sendMessage(conn, "The game started! Guess the number from 1-100:")
		fmt.Printf("Secret Number for %s is %d", username, secretNumber)

		for {
			guessStr, err := readMessage(reader)
			if err != nil {
				fmt.Println("Error reading message", err)
				return
			}
			guess, err := strconv.Atoi(guessStr)
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

func authenticateUser(conn net.Conn, reader *bufio.Reader) (string, error) {
	sendMessage(conn, "Enter Username:")
	username, err := readMessage(reader)
	if err != nil {
		return "", err
	}

	sendMessage(conn, "Enter Password:")
	password, err := readMessage(reader)
	if err != nil {
		return "", err
	}

	mu.Lock()
	defer mu.Unlock()

	encodedPassword := base64.StdEncoding.EncodeToString([]byte(password))

	if storedPassword, exists := users[username]; exists && storedPassword == password {
		if storedPassword == encodedPassword {
			sendMessage(conn, "Authentication successful!")
			return username, err
		}
		sendMessage(conn, "Authentication failed. Closing connection.")
		return " ", fmt.Errorf("Invalid Credentials")
	}
	users[username] = encodedPassword
	saveUsers()
	sendMessage(conn, "New User registered successfully!")
	return username, nil
}

func saveUsers() {
	file, err := os.Create(userFile)
	if err != nil {
		fmt.Println("Error saving users:", err)
		return
	}
	defer file.Close()

	var userList []User
	for username, password := range users {
		userList = append(userList, User{Username: username, Password: password})
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent(" ", "  ")
	err = encoder.Encode(userList)
	if err != nil {
		fmt.Println("Error Encoding users: ", err)
	}
}

func loadUsers() {
	file, err := os.Open(userFile)
	if err != nil {
		fmt.Println("No existing user file.")
		return
	}

	defer file.Close()

	var userList []User
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&userList)
	if err != nil {
		fmt.Println("Error decoding users:", err)
		return
	}

	for _, user := range userList {
		users[user.Username] = user.Password
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
