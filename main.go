package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

	pb "github.com/KoyamaSohei/pdns-grpc/proto"
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
	if host := os.Getenv("GRPC_HOST"); host != "" {
		pdnshost = host
	}
	if port := os.Getenv("GRPC_PORT"); port != "" {
		pdnsport = port
	}
	if host := os.Getenv("GPGSQL_HOST"); host != "" {
		psqlhost = host
	}
	if name := os.Getenv("GPGSQL_DBNAME"); name != "" {
		psqlname = name
	}
	if user := os.Getenv("GPGSQL_USER"); user != "" {
		psqluser = user
	}
	if pass := os.Getenv("GPGSQL_PASSWORD"); pass != "" {
		psqlpass = pass
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
	lis, err := net.Listen("tcp", pdnshost+":"+pdnsport)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("listening on %s:%s", pdnshost, pdnsport)
	s := grpc.NewServer()
	pb.RegisterPdnsServiceServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
