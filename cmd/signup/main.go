package main

import (
	"flag"

	"github.com/acheong08/nameserver/database"
	"github.com/acheong08/nameserver/models"
)

func main() {
	username := flag.String("username", "", "New username")
	passwd := flag.String("password", "", "New password")
	domain := flag.String("domain", "", "New domain")
	flag.Parse()
	if *username == "" || *passwd == "" {
		panic("Username or password missing")
	}

	store, err := database.NewStorage("127.0.0.1")
	if err != nil {
		panic(err)
	}
	err = store.DB.NewUser(models.User{
		Username: *username,
		Password: *passwd,
		Domain: *domain,
	})
	if err != nil {
		panic(err)
	}

}
