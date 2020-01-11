package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

	pb "github.com/KoyamaSohei/special-seminar-api/proto"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_auth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"go.uber.org/zap"
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

var (
	db     *sql.DB
	logger *zap.Logger
)

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
	logger.Info("psqlhost: " + psqlhost)
}

// GetDB get db
func GetDB() *sql.DB {
	if db == nil {
		dbinfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable host=%s", psqluser, psqlpass, psqlname, psqlhost)
		newDb, err := sql.Open("postgres", dbinfo)
		if err != nil {
			logger.Error("db connection error: ", zap.Error(err))
		}
		db = newDb
	}
	return db
}

func main() {
	ops := zap.NewProductionConfig()
	ops.OutputPaths = []string{"stdout"}
	logger, _ = ops.Build()
	initConfig()
	InitJWTAuth()
	lis, err := net.Listen("tcp", pdnshost+":"+pdnsport)
	if err != nil {
		logger.Error("failed to listen", zap.Error(err))
	}
	logger.Info("listening on " + pdnshost + " : " + pdnsport)
	s := grpc.NewServer(
		grpc_middleware.WithStreamServerChain(
			grpc_auth.StreamServerInterceptor(AuthHandler),
			grpc_zap.StreamServerInterceptor(zap.NewNop())),
		grpc_middleware.WithUnaryServerChain(
			grpc_auth.UnaryServerInterceptor(AuthHandler),
			grpc_zap.UnaryServerInterceptor(logger)))
	pb.RegisterPdnsServiceServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
