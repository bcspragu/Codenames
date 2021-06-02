package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/bcspragu/Codenames/cryptorand"
	"github.com/bcspragu/Codenames/sqldb"
	"github.com/bcspragu/Codenames/web"
	"github.com/gorilla/securecookie"
	"github.com/namsral/flag"

	"math/rand"
)

func main() {
	var (
		addr   = flag.String("addr", ":8080", "HTTP service address")
		dbPath = flag.String("db_path", "codenames.db", "Path to the SQLite DB file")
	)

	flag.Parse()

	r := rand.New(cryptorand.NewSource())
	db, err := sqldb.New(*dbPath, r)
	if err != nil {
		log.Fatalf("failed to initialize datastore: %v", err)
	}

	sc, err := loadKeys()
	if err != nil {
		log.Fatalf("failed to load cookie keys: %v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		db.Close()
		os.Exit(1)
	}()

	log.Printf("Server is running on %q", *addr)
	if err := http.ListenAndServe(*addr, web.New(db, r, sc)); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func loadKeys() (*securecookie.SecureCookie, error) {
	hashKey, err := loadOrGenKey("hashKey")
	if err != nil {
		return nil, err
	}

	blockKey, err := loadOrGenKey("blockKey")
	if err != nil {
		return nil, err
	}

	return securecookie.New(hashKey, blockKey), nil
}

func loadOrGenKey(name string) ([]byte, error) {
	f, err := ioutil.ReadFile(name)
	if err == nil {
		return f, nil
	}

	dat := securecookie.GenerateRandomKey(32)
	if dat == nil {
		return nil, errors.New("Failed to generate key")
	}

	err = ioutil.WriteFile(name, dat, 0777)
	if err != nil {
		return nil, errors.New("Error writing file")
	}
	return dat, nil
}
