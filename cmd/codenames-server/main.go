package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bcspragu/Codenames/sqldb"
	"github.com/bcspragu/Codenames/web"
	"github.com/namsral/flag"

	cryptorand "crypto/rand"
	"math/rand"
)

func main() {
	var (
		_      = flag.String(flag.DefaultConfigFlagname, "config", "Path to config file")
		addr   = flag.String("addr", ":8080", "HTTP service address")
		dbPath = flag.String("db_path", "codenames.db", "Path to the SQLite DB file")
	)

	rand.Seed(time.Now().Unix())
	flag.Parse()

	db, err := sqldb.New(*dbPath, rand.New(cryptoRandSource{}))
	if err != nil {
		log.Fatalf("Failed to initialize datastore: %v", err)
	}

	srv, err := web.New(db)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		db.Close()
		os.Exit(1)
	}()

	if err := http.ListenAndServe(*addr, srv); err != nil {
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
