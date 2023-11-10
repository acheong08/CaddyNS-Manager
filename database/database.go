package database

import (
	"database/sql"
	"log"

	"github.com/acheong08/nameserver/models"
	sqlx "github.com/acheong08/squealx"
	_ "github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
)

const (
	createUserTable = `
		CREATE TABLE IF NOT EXISTS users (
			username TEXT PRIMARY KEY,
			password TEXT NOT NULL,
			domain TEXT NOT NULL
		)
	`
	createServiceTable = `
		CREATE TABLE IF NOT EXISTS services (
			owner TEXT NOT NULL,
			destination TEXT NOT NULL,
			port INTEGER NOT NULL,
			dns_record_type TEXT NOT NULL,
			subdomain TEXT NOT NULL,
			forwarding INTEGER NOT NULL,
			rate_limit INTEGER NOT NULL,
			limit_by INTEGER NOT NULL,
			PRIMARY KEY (owner, subdomain, destination)
		)
	`
)

type database struct {
	db *sqlx.DB
}

func newDatabase() (*database, error) {
	db, err := sqlx.Open("sqlite", "nameserver.db")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(createUserTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(createServiceTable)
	if err != nil {
		return nil, err
	}

	return &database{db}, nil
}

func (d *database) Close() error {
	return d.db.Close()
}

func (d *database) NewUser(user models.User) error {
	tx, err := d.db.Begin()

	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			log.Printf("Rolling back transaction due to %s\n", err.Error())
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()
	// Hash password using bcrypt
	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = tx.Exec("INSERT INTO users (username, password, domain) VALUES (?, ?, ?)", user.Username, string(hashed), user.Domain)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (d *database) UserLogin(username, password string) error {
	var hashed string
	err := d.db.QueryRow("SELECT password FROM users WHERE username = ?", username).Scan(&hashed)
	if err != nil {
		return err
	}
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
}

func (d *database) GetUser(username string) (models.User, error) {
	var user models.User
	err := d.db.QueryRowx("SELECT username, domain FROM users WHERE username = ?", username).StructScan(&user)
	return user, err
}

func (d *database) GetDomainOwner(domain string) (models.User, error) {
	var user models.User
	err := d.db.QueryRowx("SELECT username, domain FROM users WHERE domain = ?", domain).StructScan(&user)
	return user, err
}

func (d *database) NewService(service models.ServiceEntry) (*sql.Tx, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec("INSERT INTO services (owner, destination, port, dns_record_type, subdomain, forwarding, rate_limit, limit_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?)", service.Owner, service.Destination, service.Port, service.DNSRecordType, service.Subdomain, service.Forwarding, service.RateLimit, service.LimitBy)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (d *database) GetService(owner, subdomain string) ([]models.ServiceEntry, error) {
	var service = make([]models.ServiceEntry, 0)
	err := d.db.Select(&service, "SELECT * FROM services WHERE owner = ? AND subdomain = ?", owner, subdomain)
	return service, err
}

func (d *database) GetServices(owner string) ([]models.ServiceEntry, error) {
	services := make([]models.ServiceEntry, 0)
	err := d.db.Select(&services, "SELECT subdomain FROM services WHERE owner = ?", owner)
	return services, err
}

func (d *database) DeleteService(owner, subdomain string) (*sql.Tx, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec("DELETE FROM services WHERE owner = ? AND subdomain = ?", owner, subdomain)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (d *database) UpdateService(service models.ServiceEntry) (*sql.Tx, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec("UPDATE services SET destination = ?, port = ?, dns_record_type = ?, forwarding = ?, rate_limit = ?, limit_by = ? WHERE owner = ? AND subdomain = ?", service.Destination, service.Port, service.DNSRecordType, service.Forwarding, service.RateLimit, service.LimitBy, service.Owner, service.Subdomain)
	if err != nil {
		return nil, err
	}
	return tx, nil
}
