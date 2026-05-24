FROM golang:alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o agent-mail .

FROM scratch
COPY --from=builder /build/agent-mail /agent-mail
ENTRYPOINT ["/agent-mail"]
