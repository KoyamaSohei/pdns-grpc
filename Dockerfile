FROM golang
WORKDIR /go/src/github.com/KoyamaSohei/pdns-grpc
ENV GO111MODULE=on
COPY . .
RUN openssl genrsa > jwtkey.rsa
RUN openssl rsa -in jwtkey.rsa -pubout > jwtkey.rsa.pub
RUN go build 
CMD ./pdns-grpc