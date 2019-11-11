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
	empty "github.com/golang/protobuf/ptypes/empty"
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

func getAccountID(ctx context.Context, tx *sql.Tx) (string, error) {
	info, err := getInfo(ctx)
	if err != nil {
		return "", err
	}
	var id string
	err = tx.QueryRowContext(ctx, "SELECT id FROM accounts WHERE email = $1;", info.Subject).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func getDomainID(ctx context.Context, tx *sql.Tx, name string, account string) (string, error) {
	var id string
	err := tx.QueryRowContext(ctx, "SELECT id FROM domains WHERE name = $1 AND account = $2;", name, account).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func updateSoa(ctx context.Context, tx *sql.Tx, origin string, account string) error {
	id, err := getDomainID(ctx, tx, origin, account)
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
	se++
	_, err = tx.ExecContext(ctx, "DELETE FROM records WHERE domain_id = $1 AND type = 'SOA';", id)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "INSERT INTO records(domain_id,name,type,content,change_date) VALUES ($1,$2,'SOA',$3,$4);", id, origin, fmt.Sprintf("%s %s %d 60 60 60 60", mname, rname, se), se)
	return err
}

func (s *server) CreateAccount(ctx context.Context, in *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	email := in.GetEmail()
	pass := in.GetPassword()
	if email == "" || pass == "" {
		return &pb.CreateAccountResponse{Status: pb.CreateAccountResponse_BadRequest}, nil
	}
	tx, err := GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.CreateAccountResponse{Status: pb.CreateAccountResponse_InternalServerError}, err
	}

	var id string
	err = tx.QueryRowContext(ctx, "SELECT id FROM accounts WHERE email = $1;", email).Scan(&id)
	if err == nil {
		_ = tx.Rollback()
		return &pb.CreateAccountResponse{Status: pb.CreateAccountResponse_AlreadyExists}, nil
	}

	_, err = tx.ExecContext(ctx, "INSERT INTO accounts(email,password) VALUES ($1,crypt($2, gen_salt('bf')));", email, pass)
	if err != nil {
		_ = tx.Rollback()
		return &pb.CreateAccountResponse{Status: pb.CreateAccountResponse_InternalServerError}, err
	}
	err = tx.Commit()
	if err != nil {
		return &pb.CreateAccountResponse{Status: pb.CreateAccountResponse_InternalServerError}, err
	}
	token, err := authInstance.GenerateJWTToken(email)
	if err != nil {
		return &pb.CreateAccountResponse{Status: pb.CreateAccountResponse_InternalServerError}, err
	}
	return &pb.CreateAccountResponse{Status: pb.CreateAccountResponse_Ok, Token: token}, nil
}

func (s *server) InitZone(ctx context.Context, in *pb.InitZoneRequest) (*pb.InitZoneResponse, error) {
	tx, err := GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.InitZoneResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	a, err := getAccountID(ctx, tx)
	if err != nil {
		return &pb.InitZoneResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	id, err := getDomainID(ctx, tx, in.GetDomain(), a)

	if err == nil {
		_, err = tx.ExecContext(ctx, "DELETE FROM records WHERE domain_id = $1;", id)
		if err != nil {
			tx.Rollback()
			return &pb.InitZoneResponse{Status: pb.ResponseStatus_InternalServerError}, err
		}
	} else {
		_, err = tx.ExecContext(ctx, "INSERT INTO domains(name,type,account) VALUES ($1,'master',$2);", in.GetDomain(), a)
	}

	if err != nil {
		tx.Rollback()
		return &pb.InitZoneResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	id, err = getDomainID(ctx, tx, in.GetDomain(), a)
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

func (s *server) RemoveZone(ctx context.Context, in *pb.RemoveZoneRequest) (*pb.RemoveZoneResponse, error) {
	tx, err := GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.RemoveZoneResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	a, err := getAccountID(ctx, tx)
	id, err := getDomainID(ctx, tx, in.GetDomain(), a)
	if err != nil {
		return &pb.RemoveZoneResponse{Status: pb.ResponseStatus_BadRequest}, nil
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM records WHERE domain_id = $1;", id)
	if err != nil {
		tx.Rollback()
		return &pb.RemoveZoneResponse{Status: pb.ResponseStatus_InternalServerError}, nil
	}
	_, err = tx.ExecContext(ctx, "DELETE FROM domains WHERE id = $1;", id)
	if err != nil {
		tx.Rollback()
		return &pb.RemoveZoneResponse{Status: pb.ResponseStatus_InternalServerError}, nil
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return &pb.RemoveZoneResponse{Status: pb.ResponseStatus_InternalServerError}, nil
	}
	return &pb.RemoveZoneResponse{Status: pb.ResponseStatus_Ok}, nil
}

func (s *server) AddRecord(ctx context.Context, in *pb.AddRecordRequest) (*pb.AddRecordResponse, error) {
	tx, err := GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.AddRecordResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	a, err := getAccountID(ctx, tx)
	if err != nil {
		return nil, err
	}
	o := in.GetOrigin()
	id, err := getDomainID(ctx, tx, o, a)
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
	err = updateSoa(ctx, tx, o, a)
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
	a, err := getAccountID(ctx, tx)
	if err != nil {
		return nil, err
	}
	id, err := getDomainID(ctx, tx, in.GetOrigin(), a)
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

func (s *server) UpdateRecord(ctx context.Context, in *pb.UpdateRecordRequest) (*pb.UpdateRecordResponse, error) {
	tx, err := GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.UpdateRecordResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	a, err := getAccountID(ctx, tx)
	if err != nil {
		return nil, err
	}
	id, err := getDomainID(ctx, tx, in.GetOrigin(), a)
	if err != nil {
		return &pb.UpdateRecordResponse{Status: pb.ResponseStatus_BadRequest}, err
	}
	t := in.GetTarget()
	c := in.GetSource()
	_, err = tx.ExecContext(ctx, "UPDATE records SET name = $1, type = $2, ttl = $3, content = $4 WHERE name = $5 AND type = $6 AND content = $7 AND domain_id = $8;",
		c.GetName(), c.GetType().String(), c.GetTtl(), c.GetContent(), t.GetName(), t.GetType().String(), t.GetContent(), id)
	if err != nil {
		tx.Rollback()
		return &pb.UpdateRecordResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return &pb.UpdateRecordResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	return &pb.UpdateRecordResponse{Status: pb.ResponseStatus_Ok}, nil
}

func (s *server) GetDomains(ctx context.Context, in *empty.Empty) (*pb.GetDomainsResponse, error) {
	tx, err := GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.GetDomainsResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	a, err := getAccountID(ctx, tx)
	if err != nil {
		return nil, err
	}
	rows, err := GetDB().QueryContext(ctx, "SELECT id,name FROM domains WHERE account = $1;", a)
	if err != nil {
		tx.Rollback()
		return &pb.GetDomainsResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	li := make([]*pb.Domain, 0, 10)
	for rows.Next() {
		item := new(pb.Domain)
		err := rows.Scan(&item.Id, &item.Name)
		if err != nil {
			tx.Rollback()
			return &pb.GetDomainsResponse{Status: pb.ResponseStatus_InternalServerError}, err
		}
		li = append(li, item)
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return &pb.GetDomainsResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	return &pb.GetDomainsResponse{Status: pb.ResponseStatus_Ok, Domains: li}, nil
}

func (s *server) GetRecords(ctx context.Context, in *pb.GetRecordsRequest) (*pb.GetRecordsResponse, error) {
	tx, err := GetDB().BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return &pb.GetRecordsResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
	a, err := getAccountID(ctx, tx)
	if err != nil {
		return nil, err
	}
	id, err := getDomainID(ctx, tx, in.GetOrigin(), a)
	if err != nil {
		return &pb.GetRecordsResponse{Status: pb.ResponseStatus_BadRequest}, err
	}
	rows, err := GetDB().QueryContext(ctx, "SELECT name,type,content,ttl FROM records WHERE domain_id = $1 AND type != 'SOA';", id)
	if err != nil {
		tx.Rollback()
		return &pb.GetRecordsResponse{Status: pb.ResponseStatus_InternalServerError}, err
	}
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
