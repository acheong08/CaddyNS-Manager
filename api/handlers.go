package api

import (
	"crypto/rand"
	"strconv"
	"strings"

	"github.com/acheong08/nameserver/caddy"
	"github.com/acheong08/nameserver/database"
	"github.com/acheong08/nameserver/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var Secret [32]byte

func init() {
	rand.Read(Secret[:])
}

func AuthMiddleware(c *gin.Context) {
	auth := c.Request.Header.Get("Authorization")
	if auth == "" {
		// Check cookie for auth
		auth, _ = c.Cookie("Authorization")
		if auth == "" {
			c.JSON(401, gin.H{"error": "Authorization header missing"})
			c.Abort()
			return
		}
	}
	// JWT
	token, err := jwt.Parse(auth, func(token *jwt.Token) (interface{}, error) {
		return Secret[:], nil
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
	storage := c.MustGet("storage").(*database.Storage)
	err := storage.DB.UserLogin(username, password)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}
	// Get user from storage
	user, err := storage.DB.GetUser(username)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"domain":   user.Domain,
	})
	tokenString, err := token.SignedString(Secret[:])
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.SetCookie("Authorization", tokenString, 0, "/", "", false, true)
	c.Header("Location", "/")
	c.JSON(302, gin.H{"token": tokenString})
}

func ServiceEntry(c *gin.Context) {
	storage := c.MustGet("storage").(*database.Storage)
	owner := c.MustGet("user").(models.User)

	if c.Request.Method == "GET" {
		subdomain := c.Query("subdomain")
		if subdomain == "" {
			// Get all services for user
			services, err := storage.DB.GetServices(owner.Username)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			for i := range services {
				if services[i].Subdomain == "" {
					services[i].Subdomain = owner.Domain
				}
			}
			c.JSON(200, services)
			return
		}
		if subdomain == "<makenew>" {
			c.JSON(200, models.ServiceEntry{
				Domain: owner.Domain,
			})
			return
		}
		subdomainInt, err := strconv.Atoi(subdomain)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid subdomain ID"})
			return
		}
		// Get service for user and domain
		service, err := storage.DB.GetService(owner.Username, subdomainInt)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		service.Domain = owner.Domain
		c.JSON(200, service)
		return
	}
	var config models.ServiceEntry
	if err := c.BindJSON(&config); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Prevent users from adding service entries for other users
	config.Owner = owner.Username
	config.Subdomain = strings.ToLower(config.Subdomain)
	var message string

	switch c.Request.Method {
	case "POST":
		if !config.IsValidFOrPost() {
			c.JSON(400, gin.H{"error": "Invalid service entry"})
			return
		}
		tx, err := storage.DB.NewService(config)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		if config.Forwarding {
			// Delete old service service entry
			// Error can be ignored since it might not exist
			caddy.RemoveHost(config.Subdomain + "." + owner.Domain)
			err = caddy.AddConfig(caddy.NewConfig(config.Subdomain+"."+owner.Domain, constructUpstream(config.Destination, config.Port)))
			if err != nil {
				tx.Rollback()
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		}
		tx.Commit()
		storage.Cache.Delete(config.Subdomain + "." + owner.Domain)
		message = "Service entry added"

	case "DELETE":
		// Remove service entry from storage
		tx, err := storage.DB.DeleteService(owner.Username, config.ID)
		if err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if config.Forwarding {
			// Update caddy
			err = caddy.RemoveHost(config.Subdomain + "." + owner.Domain)

			if err != nil {
				tx.Rollback()
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		}
		tx.Commit()
		storage.Cache.Delete(config.Subdomain + "." + owner.Domain)
		message = "Service entry removed"

	case "PATCH":
		if !config.IsValidFOrPost() {
			c.JSON(400, gin.H{"error": "Invalid service entry"})
			return
		}
		tx, err := storage.DB.UpdateService(config)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		if config.Forwarding {
			err = caddy.Update(caddy.NewConfig(
				config.Subdomain+"."+owner.Domain,
				constructUpstream(config.Destination, config.Port),
			))
			if err != nil {
				tx.Rollback()
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
		}
		tx.Commit()
		storage.Cache.Delete(config.Subdomain + "." + owner.Domain)
		message = "Service entry updated"

	default:
		c.JSON(405, gin.H{"error": "Method not allowed"})
		return

	}

	c.JSON(200, gin.H{"success": message})
	return
}

func ClearCache(c *gin.Context) {
 	storage := c.MustGet("storage").(*database.Storage)
	storage.Cache.Clear()
	c.JSON(200, gin.H{"success": "Cache cleared"})
}

func constructUpstream(dest string, port int) string {
	return dest + ":" + strconv.Itoa(port)
}
