package controllers

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	_ "github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
	"socialhive/database"
	"socialhive/helper"
	"socialhive/models"
	"strconv"
	"time"
)

var validate = validator.New()
var userCollection *mongo.Collection = database.OpenCollection(database.Client, "user-collection")

func SignUp(c *gin.Context) {
	// fetch json data and store to user
	var user models.User
	if err := c.ShouldBind(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Println(user)

	// check whether same email exists
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	filter := bson.M{"email": user.Email}
	var foundUser models.User
	err := userCollection.FindOne(ctx, filter).Decode(&foundUser)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This email is already in use"})
		return
	}

	// validate user model
	validationErr := validate.Struct(user)
	if validationErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validationErr.Error()})
		return
	}

	// hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user.Password = string(hashedPassword)

	// add timestamp
	user.CreatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))

	// add id
	user.ID = primitive.NewObjectID()

	// default dp added
	user.Dp = os.Getenv("DEFAULT_DP")

	otpNumber, err := helper.GenerateAndSendOTP(user.Email, user.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	temperaryUserCollection := database.OpenCollection(database.Client, "temp-user-collection")

	tempUser := models.TemperaryUser{
		Otp:  strconv.Itoa(otpNumber),
		User: user,
	}

	_, err = temperaryUserCollection.InsertOne(ctx, tempUser)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	//// store the user in db
	//insertedUser, err := userCollection.InsertOne(ctx, user)
	//if err != nil {
	//	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	//	return
	//}
	c.JSON(http.StatusOK, gin.H{"data": "otp sent successfully"})

}

func CreateUserByOtp(c *gin.Context) {
	type Otp struct {
		OtpNumber string `json:"otpNumber"`
	}
	var otp Otp
	if err := c.ShouldBind(&otp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"otp": otp.OtpNumber}

	tempUserCollection := database.OpenCollection(database.Client, "temp-user-collection")

	var tempUser models.TemperaryUser
	err := tempUserCollection.FindOne(ctx, filter).Decode(&tempUser)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if tempUser.Otp != otp.OtpNumber {
		c.JSON(http.StatusBadRequest, gin.H{"error": "wrong otp"})
		return
	}

	// store the user in db
	insertedUser, err := userCollection.InsertOne(ctx, tempUser.User)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": insertedUser})

}

func Login(c *gin.Context) {
	// retrieve user data form json
	var user models.User
	if err := c.ShouldBind(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// check whether email or password are empty
	if user.Email == "" || user.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email or Password is empty"})
		return
	}

	// check whether user exists
	var foundUser models.User
	filter := bson.M{"email": user.Email}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := userCollection.FindOne(ctx, filter).Decode(&foundUser)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user does not exists"})
		return
	}

	// match the passwords
	err = bcrypt.CompareHashAndPassword([]byte(foundUser.Password), []byte(user.Password))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid password"})
		return
	}

	// create jwt token
	// Create a new token object, specifying signing method and the claims
	// you would like it to contain.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.Email,
		"exp": time.Now().Add(time.Hour * 24 * 30).Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, err := token.SignedString([]byte(os.Getenv("SECRET_KEY")))

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// send jwt-token in cookie
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("token", tokenString, 3600*24*30, "/", "", false, true)

	c.JSON(http.StatusOK, gin.H{"data": foundUser})
}

func Logout(c *gin.Context) {
	// Clear the JWT token by setting the cookie with an expired time
	c.SetCookie("token", "", -1, "/", "", false, true)

	// Respond to the client with a success message
	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

func ValidateUser(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user does not exist"})
		return
	}
	userDetails := user.(models.User)

	c.JSON(http.StatusOK, gin.H{"user got validated": userDetails.Email})

}

func GetAllUsers(c *gin.Context) {
	var users []models.User
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cursor, err := userCollection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := cursor.All(ctx, &users); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users})
}

func GetUserById(c *gin.Context) {
	userId := c.Param("id")
	objectId, err := primitive.ObjectIDFromHex(userId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	filter := bson.M{"_id": objectId}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = userCollection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	type UserToSend struct {
		ID        primitive.ObjectID `json:"_id"`
		Name      string             `json:"name"`
		Email     string             `json:"email"`
		CreatedAt time.Time          `json:"createdAt"`
		Dp        string             `json:"dp"`
	}

	userToSend := UserToSend{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		Dp:        user.Dp,
	}

	c.JSON(http.StatusOK, gin.H{"data": userToSend})
}
