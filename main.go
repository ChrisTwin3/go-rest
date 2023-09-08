package main

import (
	"flag"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zalando/gin-oauth2/github"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Create struct for schema
type Person struct {
	gorm.Model
	ID     int32  `gorm:"primarykey;size:16;autoincrement"`
	Name   string `gorm:"size:24"`
	Age    int32
	Height int32
}

// Define global variables
var db *gorm.DB
var err error
var redirectUrl, credFile string

// Get all users from db
func getUsers(c *gin.Context) {
	var users = []Person{}
	if res := db.Find(&users); res.Error != nil {
		return
	}

	c.IndentedJSON(http.StatusOK, users)
}

// Get users from db which match id
func getUsersByID(c *gin.Context) {
	id := c.Param("id")
	var user Person

	if res := db.First(&user, id); res.Error != nil {
		return
	}

	c.IndentedJSON(http.StatusOK, user)
}

// Create new user in db
func postUsers(c *gin.Context) {
	var newUser Person

	if err := c.BindJSON(&newUser); err != nil {
		return
	}

	if res := db.Create(&newUser); res.Error != nil {
		return
	}

	c.IndentedJSON(http.StatusCreated, newUser)
}

// Update existing user
func putUsers(c *gin.Context) {
	var user Person
	id := c.Param("id")

	if res := db.First(&user, id); res.Error != nil {
		return
	}
	if err := c.BindJSON(&user); err != nil {
		return
	}
	if res := db.Save(&user); res.Error != nil {
		return
	}

	c.IndentedJSON(http.StatusOK, user)
}

// Handle user info returned from login
func UserInfoHandler(c *gin.Context) {
	// Initialise variables
	var (
		res github.AuthUser
		val interface{}
		ok  bool
	)

	// Get user from github
	val = c.MustGet("user")
	if res, ok = val.(github.AuthUser); !ok {
		// If no user, return that
		res = github.AuthUser{
			Name: "no User",
		}
	}

	// Display status 200, and user name
	c.JSON(http.StatusOK, gin.H{"Hello": "from private", "user": res})
}

func InitDb() {

}

func init() {
	// Parse flags into variables
	flag.StringVar(&redirectUrl, "redirect", "http://localhost:8080/auth/", "URL to be redirected to")
	flag.StringVar(&credFile, "cred-file", "clientid.github.json", "Credential JSON file")

	// Open connection to SQLite database
	db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		// If can't open, panic
		panic("failed to connect to database")
	}

	// Update schema based on Person struct
	db.AutoMigrate(&Person{})
}

func main() {
	// Parse flags
	flag.Parse()

	// Create variables for auth
	scopes := []string{
		"read:user",
	}
	secret := []byte("secret")
	sessionName := "gosession"

	// Create router and setup github auth
	router := gin.Default()
	github.Setup(redirectUrl, credFile, scopes, secret)
	router.Use(github.Session(sessionName))

	// Setup basic router endpoints
	router.GET("/ping", func(c *gin.Context) {
		c.IndentedJSON(http.StatusOK, gin.H{"message": "pong"})
	})
	router.GET("/users", getUsers)
	router.GET("/users/:id", getUsersByID)
	router.POST("/users", postUsers)
	router.PUT("/users/:id", putUsers)

	// Create login page using github default
	router.GET("/login", github.LoginHandler)

	// Create private router group, containing endpoints that require auth
	private := router.Group("/auth")
	private.Use(github.Auth())
	private.GET("/", UserInfoHandler)
	private.GET("/api", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Hello to private"})
	})

	// Run api on localhost
	router.Run("localhost:8080")
}
