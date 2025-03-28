package main

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	userFile   = "users.json"
	fileFolder = "C:\\Users\\Acer\\Documents\\NetCentric\\Net-Centric-Lab-3-main\\server\\"
)

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

	fmt.Println("Server listening on port 8080")
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Connection Error:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer func() {
		mu.Lock()
		delete(activeUsers, conn)
		mu.Unlock()
		conn.Close()
	}()

	reader := bufio.NewReader(conn)

	username, err := authenticateUser(conn, reader)
	if err != nil {
		fmt.Println("Authentication Failed:", err)
		return
	}

	fmt.Printf("%s logged in\n", username)

	mu.Lock()
	activeUsers[conn] = username
	mu.Unlock()

	for {
		sendMessage(conn, "Enter a command: 'play' to start a game, 'download:<filename>' to download a file, or 'exit' to quit:")
		command, err := readMessage(reader)
		if err != nil {
			fmt.Println("Error reading command:", err)
			return
		}

		if command == "play" {
			playGame(conn, reader, username)
		} else if strings.HasPrefix(command, "download:") {
			filename := strings.TrimPrefix(command, "download:")
			serveFile(conn, filename)
		} else if command == "exit" {
			sendMessage(conn, "Goodbye!")
			return
		} else {
			sendMessage(conn, "Invalid command. Try again.")
		}
	}
}
func GenerateVietlot(length int) []int {
	numbers := make([]int, length)
	for i := range numbers {
		numbers[i] = rand.Intn(45) + 1
	}

	return numbers
}

func playGame(conn net.Conn, reader *bufio.Reader, username string) {
	secretNumber := GenerateVietlot(6)
	sendMessage(conn, "The game started! Guess the number:")
	fmt.Printf("Secret Number for %s is %d\n", username, secretNumber)

	for {
		guessStr, err := readMessage(reader)
		if err != nil {
			fmt.Println("Error reading guess:", err)
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
			fmt.Println("Error reading command:", err)
			return
		}
		if command == "restart" {
			playGame(conn, reader, username)
			return
		} else if command == "exit" {
			sendMessage(conn, "Goodbye!")
			return
		} else {
			sendMessage(conn, "Invalid input. Type 'restart' to play again or 'exit' to quit:")
		}
	}
}

func serveFile(conn net.Conn, filename string) {
	filePath := fileFolder + filename

	// Check if file is a text file
	if !strings.HasSuffix(filename, ".txt") {
		sendMessage(conn, "Error: Only .txt files are supported for download.")
		return
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		sendMessage(conn, "Error: File not found.")
		return
	}
	defer file.Close()

	// Read the file content
	content, err := io.ReadAll(file)
	if err != nil {
		sendMessage(conn, "Error: Unable to read file.")
		return
	}

	// Send file content to the client
	sendMessage(conn, string(content))
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

	if storedPassword, exists := users[username]; exists {
		if storedPassword == encodedPassword {
			sendMessage(conn, "Authentication successful!")
			return username, nil
		}
		sendMessage(conn, "Authentication failed. Closing connection.")
		return "", fmt.Errorf("invalid credentials")
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
	encoder.SetIndent("", "  ")
	err = encoder.Encode(userList)
	if err != nil {
		fmt.Println("Error encoding users:", err)
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
	lengthBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(lengthBytes, length)

	conn.Write(lengthBytes)
	conn.Write(msgBytes)
}

func readMessage(reader *bufio.Reader) (string, error) {
	lengthBytes := make([]byte, 2)
	_, err := io.ReadFull(reader, lengthBytes)
	if err != nil {
		return "", err
	}

	length := binary.BigEndian.Uint16(lengthBytes)
	msgBytes := make([]byte, length)
	_, err = io.ReadFull(reader, msgBytes)
	if err != nil {
		return "", err
	}
	return string(msgBytes), nil
}
