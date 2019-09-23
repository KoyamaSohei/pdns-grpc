package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
)

var (
	pdnshost = "0.0.0.0"
	pdnsport = "50051"
	psqlhost = "postgres"
	psqlname = "postgres"
	psqluser = "postgres"
	psqlpass = ""
)

var db *sql.DB

func initConfig() {
	host := os.Getenv("GRPC_HOST")
	if host != "" {
		pdnshost = host
	}
	port := os.Getenv("GRPC_PORT")
	if port != "" {
		pdnsport = port
	}
	dbhost := os.Getenv("GPGSQL_HOST")
	if dbhost != "" {
		psqlhost = dbhost
	}
	name := os.Getenv("GPGSQL_DBNAME")
	if name != "" {
		psqlname = name
	}
	user := os.Getenv("GPGSQL_USER")
	if user != "" {
		psqluser = user
	}
	pass := os.Getenv("GPGSQL_PASSWORD")
	if pass != "" {
		psqlpass = ""
	}
}

func GetDB() *sql.DB {
	if db == nil {
		dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable host=%s", psqluser, psqlpass, psqlname, psqlhost)
		newDb, err := sql.Open("postgres", dbinfo)
		if err != nil {
			log.Fatalln(err)
		}
		db = newDb
	}
	return db
}

func main() {
	initConfig()
	lis, err := net.Listen("tcp", pdnshost+pdnsport)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("listening on %s:%s", pdnshost, pdnsport)
	s := grpc.NewServer()
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
