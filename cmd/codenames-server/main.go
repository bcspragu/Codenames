package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/bcspragu/Codenames/sqldb"
	"github.com/bcspragu/Codenames/web"
	"github.com/gorilla/securecookie"
	"github.com/namsral/flag"

	cryptorand "crypto/rand"
	"math/rand"
)

func main() {
	var (
		addr   = flag.String("addr", ":8080", "HTTP service address")
		dbPath = flag.String("db_path", "codenames.db", "Path to the SQLite DB file")
	)

	flag.Parse()

	r := rand.New(cryptoRandSource{})
	db, err := sqldb.New(*dbPath, r)
	if err != nil {
		log.Fatalf("failed to initialize datastore: %v", err)
	}

	sc, err := loadKeys()
	if err != nil {
		log.Fatalf("failed to load cookie keys: %w", err)
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

type cryptoRandSource struct{}

func (cryptoRandSource) Int63() int64 {
	var buf [8]byte
	_, err := cryptorand.Read(buf[:])
	if err != nil {
		panic(err)
	}
	return int64(buf[0]) |
		int64(buf[1])<<8 |
		int64(buf[2])<<16 |
		int64(buf[3])<<24 |
		int64(buf[4])<<32 |
		int64(buf[5])<<40 |
		int64(buf[6])<<48 |
		int64(buf[7]&0x7f)<<56
}

func (cryptoRandSource) Seed(int64) {}

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
