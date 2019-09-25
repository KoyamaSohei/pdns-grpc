package main

import (
	"context"
	"log"
	"net"
	"testing"
	"time"

	pb "github.com/KoyamaSohei/pdns-grpc/proto"
	"github.com/miekg/dns"
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
	log.Println(a.String())
}
