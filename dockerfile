# syntax=docker/dockerfile:1

FROM golang:1.19 as builder

WORKDIR /usr/src/app

COPY go.mod ./
COPY go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN go build -v -o /bai1 .

WORKDIR /app

FROM debian:stretch-slim AS final

# Import the compiled executable from the first stage.
COPY --from=builder /bai1 /app/bai1

# Expose port 3000 to our application
EXPOSE 3000

# Run the compiled binary.
ENTRYPOINT ["/app/bai1"]