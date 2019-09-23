FROM golang
WORKDIR /go/src/github.com/KoyamaSohei/pdns-grpc
ENV GO111MODULE=on
COPY . .
RUN go build 
CMD ./pdns-grpc