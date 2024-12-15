package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"net/http"
	"socialhive/database"
	"socialhive/helper"
	"socialhive/models"
	"sync"
	"time"
)

type Server struct {
	conns map[*websocket.Conn]bool
	email map[string]*websocket.Conn
	mut   sync.Mutex
}

func NewServer() *Server {
	return &Server{
		conns: make(map[*websocket.Conn]bool),
		email: make(map[string]*websocket.Conn),
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins, modify as needed for security
	},
}

func (s *Server) HandleWS(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println("Failed to upgrade connection:", err)
		return
	}
	defer func(conn *websocket.Conn) {
		err := conn.Close()
		if err != nil {
			log.Fatal("Failed to close connection:", err)
			return
		}
	}(conn)

	fmt.Println("New incoming connection from client:", conn.RemoteAddr())

	email := helper.ExtractEmail(c)

	s.mut.Lock()
	s.conns[conn] = true
	s.email[email] = conn
	s.mut.Unlock()

	s.readLoop(conn, email, c)

	s.mut.Lock()
	delete(s.conns, conn)
	s.mut.Unlock()
}

func (s *Server) readLoop(conn *websocket.Conn, senderEmail string, c *gin.Context) {
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				fmt.Println("Connection closed normally")
				break
			}
			fmt.Println("Read error:", err)
			break
		}
		s.broadcast(conn, c, msg, senderEmail)
	}
}

func (s *Server) broadcast(senderConnection *websocket.Conn, c *gin.Context, msg []byte, senderEmail string) {

	type Message struct {
		Action string `json:"action"`
		To     string `json:"to"`
		Msg    string `json:"msg"`
	}
	var parsedMessage Message
	err := json.Unmarshal(msg, &parsedMessage)
	email := parsedMessage.To

	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	s.mut.Lock()
	defer s.mut.Unlock()

	// extract recipient connection

	recipientConnection, recipientExists := s.email[email]

	sender, _ := helper.GetUserByEmail(senderEmail)
	recipient, err := helper.GetUserByEmail(email)
	msgContent := parsedMessage.Msg

	var newMsg models.Message

	if parsedMessage.Action == "send" {
		newMsg = models.Message{
			ID:        primitive.NewObjectID(),
			Sender:    sender,
			Recipient: recipient,
			Content:   string(msgContent),
			Timestamp: time.Now(),
			Status:    "sent",
		}
		// save the message in database

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		msgCollection := database.OpenCollection(database.Client, "message-collection")

		_, err = msgCollection.InsertOne(ctx, newMsg)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		messageCollection := database.OpenCollection(database.Client, "message-collection")
		objectId, err := primitive.ObjectIDFromHex(parsedMessage.Msg)
		if err != nil {
			return
		}
		var message models.Message
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err = msgCollection.FindOne(ctx, bson.M{"_id": objectId}).Decode(&message)

		// delete message from database

		_, err = messageCollection.DeleteOne(ctx, bson.M{"_id": objectId})
		if err != nil {
			fmt.Println("Error deleting message:", err)
			return
		}

		newMsg = message
	}

	type MsgToBroadCast struct {
		Action          string         `json:"action"`
		MesssageContent models.Message `json:"message_content"`
	}

	msgToBroadCast := MsgToBroadCast{
		Action:          parsedMessage.Action,
		MesssageContent: newMsg,
	}

	// Convert struct to JSON string
	msgJson, err := json.Marshal(msgToBroadCast)
	if err != nil {
		fmt.Println("Error marshalling message:", err)
		return
	}
	if recipientExists {
		go s.sendMessage(recipientConnection, string(msgJson))
	}

	// send same message to the sender
	go s.sendMessage(senderConnection, string(msgJson))

	// send message user does not exist if user is not in database
	if err != nil {
		go s.sendMessage(senderConnection, "user does not exist")
		return
	}

}

func (s *Server) sendMessage(conn *websocket.Conn, message string) {
	if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
		fmt.Println("Write error:", err)
		s.mut.Lock()
		delete(s.conns, conn)
		s.mut.Unlock()
	}
}
