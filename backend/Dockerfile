FROM golang:1.25-alpine AS builder

WORKDIR /build/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /build/server ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/server ./server

COPY source.xlsx ./source.xlsx

EXPOSE 8080
ENV ADDR=:8080
ENV DATA_FILE=/app/source.xlsx

CMD ["./server"]
