# Этап сборки: используем полный образ Go только для компиляции,
# чтобы не тащить тулчейн в финальный образ.
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 даёт статически слинкованный бинарник, который
# можно запускать в scratch/alpine без зависимостей от libc.
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/server ./cmd/server

# Финальный этап: пустой образ с одним бинарником - минимальный размер и площадь атаки.
FROM alpine:3.20

RUN apk add --no-cache ca-certificates wget && \
    adduser -D -g '' appuser

COPY --from=builder /bin/server /bin/server

USER appuser

EXPOSE 8080

ENTRYPOINT ["/bin/server"]
