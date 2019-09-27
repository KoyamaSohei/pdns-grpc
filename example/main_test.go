package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"
	"time"

	pb "github.com/KoyamaSohei/pdns-grpc/proto"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// test on ./example/docker-compose up.

func TestInitZone(t *testing.T) {
	conn, err := grpc.Dial("0.0.0.0:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewPdnsServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.InitZone(ctx, &pb.InitZoneRequest{Domain: "example.com", Account: "testuser"})
	if err != nil {
		log.Fatal(err)
	}
	if status := r.GetStatus(); status != pb.ResponseStatus_Ok {
		log.Println(status)
		return
	}
	address, err := net.ResolveIPAddr("ip", "pdns")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(address.IP.String())
	cl := dns.Client{}
	m := dns.Msg{}
	m.SetQuestion("example.com.", dns.TypeSOA)
	res, _, err := cl.Exchange(&m, address.IP.String()+":53")
	if err != nil {
		log.Fatal(err)
	}
	a := res.Answer[0].(*dns.SOA)
	log.Println(a.String())
}

func TestAddRecord(t *testing.T) {
	conn, err := grpc.Dial("0.0.0.0:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewPdnsServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = c.InitZone(ctx, &pb.InitZoneRequest{Domain: "example2.com", Account: "testuser"})
	r, err := c.AddRecord(ctx, &pb.AddRecordRequest{Name: "example2.com", Origin: "example2.com", Account: "testuser", Type: pb.RRType_A, Ttl: 3500, Content: "21.21.21.21"})
	if err != nil {
		log.Fatal(err)
	}
	if status := r.GetStatus(); status != pb.ResponseStatus_Ok {
		log.Println(status)
		return
	}
	time.Sleep(time.Second)
	address, err := net.ResolveIPAddr("ip", "pdns")
	cl := dns.Client{}
	m := dns.Msg{}
	m.SetQuestion("example2.com.", dns.TypeA)
	res, _, err := cl.Exchange(&m, address.IP.String()+":53")
	if err != nil {
		log.Fatal(err)
	}
	if len(res.Answer) == 0 {
		log.Println(res)
		log.Fatalln("record not found")
	}
	a := res.Answer[0].(*dns.A)
	assert.Equal(t, a.A.String(), "21.21.21.21")
}

func TestGetRecords(t *testing.T) {
	conn, err := grpc.Dial("0.0.0.0:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewPdnsServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = c.InitZone(ctx, &pb.InitZoneRequest{Domain: "example3.com", Account: "testuser"})
	_, err = c.AddRecord(ctx, &pb.AddRecordRequest{Name: "example3.com", Origin: "example3.com", Account: "testuser", Type: pb.RRType_A, Ttl: 3500, Content: "11.11.11.11"})
	_, err = c.AddRecord(ctx, &pb.AddRecordRequest{Name: "sub.example3.com", Origin: "example3.com", Account: "testuser", Type: pb.RRType_A, Ttl: 3500, Content: "22.22.22.22"})
	r, err := c.GetRecords(ctx, &pb.GetRecordsRequest{Origin: "example3.com", Account: "testuser"})
	assert.Equal(t, len(r.GetRecords()), 2)
}

func TestRemoveRecord(t *testing.T) {
	conn, err := grpc.Dial("0.0.0.0:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewPdnsServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = c.InitZone(ctx, &pb.InitZoneRequest{Domain: "example4.com", Account: "testuser"})
	_, err = c.AddRecord(ctx, &pb.AddRecordRequest{Name: "example4.com", Origin: "example4.com", Account: "testuser", Type: pb.RRType_A, Ttl: 3500, Content: "11.11.11.11"})
	_, err = c.RemoveRecord(ctx, &pb.RemoveRecordRequest{Name: "example4.com", Origin: "example4.com", Account: "testuser", Type: pb.RRType_A, Content: "11.11.11.11"})
	r, err := c.GetRecords(ctx, &pb.GetRecordsRequest{Origin: "example4.com", Account: "testuser"})
	assert.Equal(t, len(r.GetRecords()), 0)
}

func TestUpdateRecord(t *testing.T) {
	conn, err := grpc.Dial("0.0.0.0:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewPdnsServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = c.InitZone(ctx, &pb.InitZoneRequest{Domain: "example5.com", Account: "testuser"})
	_, err = c.AddRecord(ctx, &pb.AddRecordRequest{Name: "example5.com", Origin: "example5.com", Account: "testuser", Type: pb.RRType_A, Ttl: 3500, Content: "11.11.11.11"})
	_, err = c.UpdateRecord(ctx,
		&pb.UpdateRecordRequest{
			Origin: "example5.com", Account: "testuser",
			Target: &pb.UpdateRecordRequest_Target{Name: "example5.com", Type: pb.RRType_A, Content: "11.11.11.11"},
			Source: &pb.UpdateRecordRequest_Source{Name: "example5.com", Type: pb.RRType_A, Content: "22.22.22.22", Ttl: 9999}})
	fmt.Println(err)
	r, err := c.GetRecords(ctx, &pb.GetRecordsRequest{Origin: "example5.com", Account: "testuser"})
	assert.Equal(t, len(r.GetRecords()), 1)
	assert.Equal(t, r.GetRecords()[0].Content, "22.22.22.22")
}

func TestRemoveZone(t *testing.T) {
	conn, err := grpc.Dial("0.0.0.0:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewPdnsServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = c.InitZone(ctx, &pb.InitZoneRequest{Domain: "example6.com", Account: "testuser"})
	_, err = c.AddRecord(ctx, &pb.AddRecordRequest{Name: "example6.com", Origin: "example6.com", Account: "testuser", Type: pb.RRType_A, Ttl: 3500, Content: "33.33.33.33"})
	r, err := c.GetRecords(ctx, &pb.GetRecordsRequest{Origin: "example6.com", Account: "testuser"})
	assert.Equal(t, len(r.GetRecords()), 1)
	assert.Equal(t, r.GetRecords()[0].Content, "33.33.33.33")
	_, err = c.RemoveZone(ctx, &pb.RemoveZoneRequest{Domain: "example6.com", Account: "testuser"})
	r, err = c.GetRecords(ctx, &pb.GetRecordsRequest{Origin: "example6.com", Account: "testuser"})
	assert.Equal(t, len(r.GetRecords()), 0)
}
