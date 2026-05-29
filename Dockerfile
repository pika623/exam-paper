# syntax=docker/dockerfile:1

FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/exam-paper ./cmd

FROM debian:bookworm-slim

WORKDIR /app
COPY --from=builder /out/exam-paper /app/exam-paper
COPY static /app/static
COPY README.md /app/README.md

RUN mkdir -p /app/data /app/演出协会官方题库题目 /app/演出协会模拟考 && chmod +x /app/exam-paper

EXPOSE 16666
VOLUME ["/app/data", "/app/演出协会官方题库题目", "/app/演出协会模拟考"]

ENTRYPOINT ["/app/exam-paper"]
CMD ["--host", "0.0.0.0", "--port", "16666"]

