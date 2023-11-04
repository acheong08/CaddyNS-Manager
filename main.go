package main

import (
	"flag"
	"fmt"

	"github.com/acheong08/nameserver/api"
	"github.com/acheong08/nameserver/database"
	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
)

func main() {
	dnsAddr := flag.String("dns-addr", ":53", "DNS listen address")
	httpAddr := flag.String("http-addr", ":8080", "HTTP listen address")
	publicIP := flag.String("public-ip", "127.0.0.1", "Public IP address")
	flag.Parse()

	storage, err := database.NewStorage(*publicIP)
	if err != nil {
		panic(fmt.Errorf("Failed to start storage: %s\n", err.Error()))
	}
	defer storage.Close()

	go func(dnsAddr *string, storage *database.Storage) {
		server := &dns.Server{Addr: *dnsAddr, Net: "udp", ReusePort: true, TsigSecret: nil}
		err := server.ListenAndServe()
		if err != nil {
			panic(fmt.Errorf("Failed to start DNS server: %s\n", err.Error()))
		}
	}(dnsAddr, storage)

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		// Add storage to context
		c.Set("storage", storage)
	})
	router.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})
	router.POST("/login", api.Login)
	authNeeded := router.Group("/api")
	authNeeded.Use(api.AuthMiddleware)
	authNeeded.GET("/service", api.ServiceEntry)
	authNeeded.POST("/service", api.ServiceEntry)
	authNeeded.DELETE("/service", api.ServiceEntry)
	authNeeded.PATCH("/service", api.ServiceEntry)
	router.Run(*httpAddr)

}
