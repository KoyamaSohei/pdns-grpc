FROM golang
WORKDIR /go/src/github.com/KoyamaSohei/special-seminar-api
ENV GO111MODULE=on
COPY . .
RUN openssl genrsa > jwtkey.rsa
RUN openssl rsa -in jwtkey.rsa -pubout > jwtkey.rsa.pub
RUN go build 
CMD ./special-seminar-api