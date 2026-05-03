FROM golang:1.25-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the compiled binary
COPY --from=builder /build/server .

# Copy the Excel data file
COPY source.xlsx .

EXPOSE 8080

CMD ["./server"]
