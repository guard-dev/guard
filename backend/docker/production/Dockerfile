FROM golang:1.23

WORKDIR /app

COPY . .

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /guard-gql-server server.go

EXPOSE 80

ENTRYPOINT [ "/guard-gql-server" ]
