FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o agent-mail .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/agent-mail /agent-mail
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/agent-mail", "--db-path", "/data/agent-mail.db"]
