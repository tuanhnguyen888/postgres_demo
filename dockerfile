# syntax=docker/dockerfile:1

FROM golang:1.19

WORKDIR /usr/src/app

COPY go.mod ./
COPY go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN go build -v -o /bai1

EXPOSE 3000

CMD ["/bai1"]