package main

import (
	"embed"
	"flag"
	"fmt"
	"log"

	"github.com/acheong08/nameserver/api"
	"github.com/acheong08/nameserver/database"
	"github.com/acheong08/nameserver/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/miekg/dns"
)

//go:embed static/*
var staticEmbed embed.FS

func main() {
	dnsAddr := flag.String("dns-addr", ":5553", "DNS listen address")
	httpAddr := flag.String("http-addr", ":8080", "HTTP listen address")
	publicIP := flag.String("public-ip", "127.0.0.1", "Public IP address")
	debug := flag.Bool("debug", false, "Debug mode")
	flag.Parse()

	storage, err := database.NewStorage(*publicIP)
	if err != nil {
		panic(fmt.Errorf("Failed to start storage: %s\n", err.Error()))
	}
	defer storage.DB.Close()

	go func(dnsAddr *string, storage *database.Storage) {
		server := &dns.Server{Addr: *dnsAddr, Net: "udp", ReusePort: true, TsigSecret: nil}
		server.Handler = dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
			m := new(dns.Msg)
			m.SetReply(r)
			m.Authoritative = true
			m.RecursionAvailable = true

			qType := dns.TypeToString[r.Question[0].Qtype]
			qName := r.Question[0].Name
			dnsRecords := storage.GetDNS(qName)
			if dnsRecords == nil {
				m.SetRcode(r, dns.RcodeNameError)
				w.WriteMsg(m)
				return
			}
			for _, dnsRecord := range dnsRecords {
				if dnsRecord.RecordType == qType || dnsRecord.RecordType == "CNAME" {
					rr, err := dns.NewRR(fmt.Sprintf("%s %d IN %s %s", qName, 60, dnsRecord.RecordType, dnsRecord.Dest))
					if err != nil {
						fmt.Println(fmt.Errorf("Failed to create RR: %s\n", err.Error()))
						continue
					}
					m.Answer = append(m.Answer, rr)
				}
			}
			w.WriteMsg(m)
		})
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
	router.GET("/login.html", func(ctx *gin.Context) {
		// Serve login.html
		login, err := staticEmbed.ReadFile("static/login.html")
		if err != nil {
			ctx.String(500, err.Error())
			return
		}
		ctx.Data(200, "text/html", login)
	})
	router.GET("/", func(c *gin.Context) {
		// Check for auth cookie
		cookie, err := c.Cookie("Authorization")
		if err != nil || cookie == "" {
			c.Redirect(302, "/login.html")
			return
		}
		// JWT
		token, err := jwt.Parse(cookie, func(token *jwt.Token) (interface{}, error) {
			return api.Secret[:], nil
		})
		if err != nil {
			c.Redirect(302, "/login.html")
			return
		}
		_, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.Redirect(302, "/login.html")
			return
		}
		// Serve index.html
		index, err := staticEmbed.ReadFile("static/index.html")
		if err != nil {
			c.String(500, err.Error())
			return
		}
		c.Data(200, "text/html", index)
	})
	router.GET("/htmj.js", func(ctx *gin.Context) {
		// Serve index.html
		index, _ := staticEmbed.ReadFile("static/htmj.js")
		ctx.Data(200, "text/javascript", index)
	})
	authNeeded := router.Group("/api")
	if !*debug {
		authNeeded.Use(api.AuthMiddleware)
	} else {
		err := storage.DB.NewUser(models.User{
			Username: "admin",
			Password: "admin",
			Domain:   "example.com",
		})
		if err != nil {
			log.Println(err)
		}
		authNeeded.Use(func(c *gin.Context) {
			// Set user to admin
			c.Set("user", models.User{
				Username: "admin",
				Domain:   "example.com",
			})
		})
	}
	authNeeded.GET("/service", api.ServiceEntry)
	authNeeded.POST("/service", api.ServiceEntry)
	authNeeded.DELETE("/service", api.ServiceEntry)
	authNeeded.PATCH("/service", api.ServiceEntry)

	authNeeded.POST("/cache/clear", api.ClearCache)

	router.Run(*httpAddr)

}
