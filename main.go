package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

type application struct {
	Port   int
	Origin string
	Cache  *Cache
}

func main() {
	port := flag.Int("port", 8080, "Specify the listening port for the proxy server")
	origin := flag.String("origin", "", "Specify the URL of the target server")
	dsn := flag.String("dsn", "web:pass@/cacheProxy?parseTime=true", "MySQL data source name")
	flag.Parse()

	if *origin == "" {
		fmt.Println("Please specify the --origin parameter, e.g., --origin http://example.com")
		os.Exit(1)
	}

	DB, err := openDB(*dsn)
	if err != nil {
		log.Fatal(err)
	}

	app := &application{
		Port:   *port,
		Origin: *origin,
	}

	app.Cache.newCache(DB)

	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(app.Port),
		Handler: app.Routes(),
	}

	fmt.Printf("Starting proxy server on port: %d, target server: %s\n", *port, *origin)
	err = srv.ListenAndServe()
	log.Fatal(err)
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Println("Error opening database:", err)
		return nil, err
	}

	if err = db.Ping(); err != nil {
		log.Println("Error pinging database: ", err)
		return nil, err
	}

	log.Println("Database connection successful")
	return db, nil
}
