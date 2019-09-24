package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	pb "github.com/KoyamaSohei/pdns-grpc/proto"
)

type server struct{}

var (
	mname string = os.Getenv("SOA_MNAME")
	rname string = os.Getenv("SOA_RNAME")
)

func getDomainId(tx *sql.Tx, ctx context.Context, name string, account string) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, "SELECT id FROM domains WHERE name = '$1' AND account = '$2';", name, account).Scan(&id)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	return id, nil
}

func (s *server) InitZone(ctx context.Context, in *pb.InitZoneRequest) (*pb.InitZoneResponse, error) {
	tx, err := GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.InitZoneResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO domains(name,type,account) VALUES ('$1','master','$2');", in.GetDomain(), in.GetAccount())
	if err != nil {
		tx.Rollback()
		return &pb.InitZoneResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	id, err := getDomainId(tx, ctx, in.GetDomain(), in.GetAccount())
	if err != nil {
		tx.Rollback()
		return &pb.InitZoneResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO records(domain_id,name,type,content,change_date) VALUES ($1,'$2','SOA','$3 $4 $5 60 60 60 60',$5);", id, in.GetDomain(), mname, rname, time.Now().Unix()/1000)
	if err != nil {
		tx.Rollback()
		return &pb.InitZoneResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return &pb.InitZoneResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	return &pb.InitZoneResponse{Status: pb.ResponseStatus_Ok}, nil

}
