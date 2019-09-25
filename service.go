package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	pb "github.com/KoyamaSohei/pdns-grpc/proto"
	_ "github.com/lib/pq"
)

type server struct{}

var (
	mname string = os.Getenv("SOA_MNAME")
	rname string = os.Getenv("SOA_RNAME")
)

func genSerial() int {
	return (int)(time.Now().Unix() % 10000)
}

func getDomainId(tx *sql.Tx, ctx context.Context, name string, account string) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, "SELECT id FROM domains WHERE name = $1 AND account = $2;", name, account).Scan(&id)
	if err != nil {
		log.Println("get domain id: " + name)
		log.Println(err)
		return "", err
	}
	return id, nil
}

func (s *server) InitZone(ctx context.Context, in *pb.InitZoneRequest) (*pb.InitZoneResponse, error) {
	tx, err := GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.InitZoneResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	id, err := getDomainId(tx, ctx, in.GetDomain(), in.GetAccount())
	if err == nil {
		_, err = tx.ExecContext(ctx, "DELETE FROM records WHERE domain_id = $1;", id)
		if err != nil {
			tx.Rollback()
			return &pb.InitZoneResponse{Status: pb.ResponseStatus_InternalServerError}, err
		}
	} else {
		_, err = tx.ExecContext(ctx, "INSERT INTO domains(name,type,account) VALUES ($1,'master',$2);", in.GetDomain(), in.GetAccount())
	}

	if err != nil {
		tx.Rollback()
		return &pb.InitZoneResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	id, err = getDomainId(tx, ctx, in.GetDomain(), in.GetAccount())
	if err != nil {
		tx.Rollback()
		return &pb.InitZoneResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	se := genSerial()
	_, err = tx.ExecContext(ctx, "INSERT INTO records(domain_id,name,type,content,change_date) VALUES ($1,$2,'SOA',$3,$4);", id, in.GetDomain(), fmt.Sprintf("%s %s %d 60 60 60 60", mname, rname, se), se)
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
