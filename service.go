package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	pb "github.com/KoyamaSohei/pdns-grpc/proto"
	_ "github.com/lib/pq"
)

type server struct{}

const (
	defTTL = 3600
)

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

func updateSoa(tx *sql.Tx, ctx context.Context, origin string, account string) error {
	id, err := getDomainId(tx, ctx, origin, account)
	var c string
	err = tx.QueryRowContext(ctx, "SELECT content FROM records WHERE type = 'SOA' AND domain_id = $1;", id).Scan(&c)
	if err != nil {
		log.Println("update soa: " + origin)
		log.Println(err)
		return err
	}
	r := strings.Split(c, " ")
	if len(r) != 7 {
		return errors.New("soa record is invalid")
	}
	mname := r[0]
	rname := r[1]
	se, err := strconv.Atoi(r[2])
	if err != nil {
		se = genSerial()
	}
	se += 1
	_, err = tx.ExecContext(ctx, "DELETE FROM records WHERE domain_id = $1 AND type = 'SOA';", id)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO records(domain_id,name,type,content,change_date) VALUES ($1,$2,'SOA',$3,$4);", id, origin, fmt.Sprintf("%s %s %d 60 60 60 60", mname, rname, se), se)
	return err
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

func (s *server) AddRecord(ctx context.Context, in *pb.AddRecordRequest) (*pb.AddRecordResponse, error) {
	tx, err := GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.AddRecordResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	a := in.GetAccount()
	o := in.GetOrigin()
	id, err := getDomainId(tx, ctx, o, a)
	if err != nil {
		tx.Rollback()
		return &pb.AddRecordResponse{Status: pb.ResponseStatus_BadRequest}, err
	}
	ttl := in.GetTtl()
	if ttl == 0 {
		ttl = defTTL
	}
	se := genSerial()
	_, err = tx.ExecContext(ctx, "INSERT INTO records(domain_id,name,type,content,change_date,ttl) VALUES ($1,$2,$3,$4,$5,$6);", id, in.GetName(), in.GetType().String(), in.GetContent(), se, ttl)
	if err != nil {
		tx.Rollback()
		return &pb.AddRecordResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	err = updateSoa(tx, ctx, o, a)
	if err != nil {
		tx.Rollback()
		return &pb.AddRecordResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return &pb.AddRecordResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	return &pb.AddRecordResponse{Status: pb.ResponseStatus_Ok}, nil

}

func (s *server) RemoveRecord(ctx context.Context, in *pb.RemoveRecordRequest) (*pb.RemoveRecordResponse, error) {
	tx, err := GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.RemoveRecordResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	id, err := getDomainId(tx, ctx, in.GetOrigin(), in.GetAccount())
	if err != nil {
		return &pb.RemoveRecordResponse{Status: pb.ResponseStatus_BadRequest}, err
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM records WHERE domain_id = $1 AND name = $2 AND type = $3 AND content = $4", id, in.GetName(), in.GetType().String(), in.GetContent())
	if err != nil {
		tx.Rollback()
		return &pb.RemoveRecordResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	err = tx.Commit()
	if err != nil {
		return &pb.RemoveRecordResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	return &pb.RemoveRecordResponse{Status: pb.ResponseStatus_Ok}, nil
}

func (s *server) GetRecords(ctx context.Context, in *pb.GetRecordsRequest) (*pb.GetRecordsResponse, error) {
	tx, err := GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.GetRecordsResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	id, err := getDomainId(tx, ctx, in.GetOrigin(), in.GetAccount())
	if err != nil {
		return &pb.GetRecordsResponse{Status: pb.ResponseStatus_BadRequest}, err
	}
	rows, err := GetDB().QueryContext(ctx, "SELECT name,type,content,ttl FROM records WHERE domain_id = $1 AND type != 'SOA';", id)
	li := make([]*pb.Record, 0, 10)
	for rows.Next() {
		item := new(pb.Record)
		err := rows.Scan(&item.Name, &item.Type, &item.Content, &item.Ttl)
		if err != nil {
			tx.Rollback()
			return &pb.GetRecordsResponse{Status: pb.ResponseStatus_InternalServerError}, err
		}
		li = append(li, item)
	}

	err = tx.Commit()
	if err != nil {
		return &pb.GetRecordsResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	return &pb.GetRecordsResponse{Status: pb.ResponseStatus_Ok, Records: li}, nil
}
