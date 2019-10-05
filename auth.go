package main

import (
	"bufio"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"log"
	"os"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc/metadata"
)

type JWTAuth struct {
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

var authInstance *JWTAuth = nil

func InitJWTAuth() *JWTAuth {
	if authInstance == nil {
		authInstance = &JWTAuth{
			PrivateKey: getPrivateKey(),
			PublicKey:  getPublicKey(),
		}
	}

	return authInstance
}

func getPrivateKey() *rsa.PrivateKey {
	privateKeyFile, err := os.Open("jwtkey.rsa")
	if err != nil {
		panic(err)
	}

	pemfileinfo, _ := privateKeyFile.Stat()
	var size int64 = pemfileinfo.Size()
	pembytes := make([]byte, size)

	buffer := bufio.NewReader(privateKeyFile)
	_, err = buffer.Read(pembytes)

	data, _ := pem.Decode([]byte(pembytes))

	privateKeyFile.Close()

	privateKeyImported, err := x509.ParsePKCS1PrivateKey(data.Bytes)

	if err != nil {
		panic(err)
	}

	return privateKeyImported
}

func getPublicKey() *rsa.PublicKey {
	publicKeyFile, err := os.Open("jwtkey.rsa.pub")
	if err != nil {
		panic(err)
	}

	pemfileinfo, _ := publicKeyFile.Stat()
	var size int64 = pemfileinfo.Size()
	pembytes := make([]byte, size)

	buffer := bufio.NewReader(publicKeyFile)
	_, err = buffer.Read(pembytes)

	data, _ := pem.Decode([]byte(pembytes))

	publicKeyFile.Close()

	publicKeyImported, err := x509.ParsePKIXPublicKey(data.Bytes)

	if err != nil {
		panic(err)
	}

	rsaPub, ok := publicKeyImported.(*rsa.PublicKey)

	if !ok {
		panic(err)
	}

	return rsaPub
}

func (auth *JWTAuth) GenerateJWTToken(id string) (string, error) {
	token := jwt.New(jwt.SigningMethodRS512)
	token.Claims = jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * time.Duration(3600)).Unix(),
		"iat": time.Now().Unix(),
		"sub": id,
	}
	tokenString, err := token.SignedString(auth.PrivateKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

type contextKey string

const tokenContextKey contextKey = "token"

func SetToken(parents context.Context, t string) context.Context {
	return context.WithValue(parents, tokenContextKey, t)
}

func GetToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok || len(md.Get("token")) != 1 {
		return "", errors.New("cannot get metadata from incoming context")
	}
	token := md.Get("token")[0]
	return token, nil
}

type JwtInfo struct {
	ExpiresAt int64  `json:"exp,omitempty"`
	IssuedAt  int64  `json:"iat,omitempty"`
	Subject   string `json:"sub,omitempty"`
	jwt.StandardClaims
}

func getInfoFromToken(token string) (*JwtInfo, error) {
	s := strings.Split(token, ".")
	if len(s) != 3 {
		return nil, errors.New("token invalid")
	}
	b, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(s[1])
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var info JwtInfo
	err = json.Unmarshal(b, &info)
	if err != nil {
		log.Println("info")
		log.Println(info)
		return nil, err
	}
	return &info, nil
}

func getInfo(ctx context.Context) (*JwtInfo, error) {
	token, err := GetToken(ctx)
	if err != nil {
		return nil, err
	}
	info, err := getInfoFromToken(token)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func AuthHandler(ctx context.Context) (context.Context, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	v := md.Get("token")
	if len(v) != 1 {
		return ctx, nil
	}
	token, err := jwt.Parse(v[0], func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, errors.New("error on parsing jwt")
		}
		return authInstance.PublicKey, nil
	})
	if err != nil {
		return nil, err
	}

	newCtx := context.WithValue(ctx, "token", token)
	return newCtx, nil
}
