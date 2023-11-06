package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/acheong08/nameserver/api"
	"github.com/acheong08/nameserver/database"
	"github.com/acheong08/nameserver/models"
	"github.com/gin-gonic/gin"
	"github.com/miekg/dns"
)

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
	defer storage.Close()

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
				if dnsRecord.RecordType == qType {
					rr, err := dns.NewRR(fmt.Sprintf("%s %d IN %s %s", qName, 60, qType, dnsRecord.Dest))
					if err != nil {
						panic(fmt.Errorf("Failed to create RR: %s\n", err.Error()))
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
	authNeeded := router.Group("/api")
	if !*debug {
		authNeeded.Use(api.AuthMiddleware)
	} else {
		err := storage.NewUser(models.User{
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
	router.Run(*httpAddr)

}
