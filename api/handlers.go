package api

import (
	"crypto/rand"

	"github.com/acheong08/nameserver/database"
	"github.com/acheong08/nameserver/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var secret [32]byte

func init() {
	rand.Read(secret[:])
}

func AuthMiddleware(c *gin.Context) {
	auth := c.Request.Header.Get("Authorization")
	if auth == "" {
		c.JSON(401, gin.H{"error": "Authorization header missing"})
		c.Abort()
		return
	}
	// JWT
	token, err := jwt.Parse(auth, func(token *jwt.Token) (interface{}, error) {
		return secret[:], nil
	})
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		c.Abort()
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(401, gin.H{"error": "Invalid token"})
		c.Abort()
		return
	}

	// Convert claims to user struct
	user := models.User{
		Username: claims["username"].(string),
		Domain:   claims["domain"].(string),
	}
	// Add user to context
	c.Set("user", user)

	c.Next()
}

func Login(c *gin.Context) {
	// Get username and password from form
	var username, password string = c.PostForm("username"), c.PostForm("password")
	if username == "" || password == "" {
		c.JSON(400, gin.H{"error": "Username and password required"})
		return
	}
	// Get storage from context
	storage := c.MustGet("storage").(database.Storage)
	err := storage.UserLogin(username, password)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	// Get user from storage
	user, err := storage.GetUser(username)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"domain":   user.Domain,
	})
	tokenString, err := token.SignedString(secret[:])
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"token": tokenString})
}

func ServiceEntry(c *gin.Context) {
	storage := c.MustGet("storage").(database.Storage)
	owner := c.MustGet("user").(models.User)

	if c.Request.Method == "GET" {
		domain := c.Query("domain")
		if domain == "" {
			// Get all services for user
			services, err := storage.GetServices(owner.Username)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, services)
			return
		}
		// Get service for user and domain
		service, err := storage.GetService(owner.Username, domain)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, service)
	}
	var config models.ServiceEntry
	if err := c.BindJSON(&config); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Prevent users from adding service entries for other users
	config.Owner = owner.Username

	var err error

	switch c.Request.Method {
	case "POST":
		// Add service entry to storage
		err = storage.NewService(config)

	case "DELETE":
		// Remove service entry from storage
		err = storage.DeleteService(owner.Username, config.Subdomain)

	case "PATCH":
		err = storage.UpdateService(config)

	default:
		c.JSON(405, gin.H{"error": "Method not allowed"})
		return

	}
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": "Service entry removed"})
	return
}
