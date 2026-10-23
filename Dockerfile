FROM golang:1.27 AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -o /leaderboard .

FROM alpine:3.24
RUN adduser -D -u 10001 app
COPY --from=build /leaderboard /usr/local/bin/leaderboard
ENV LISTEN_ADDR=:8080
EXPOSE 8080
USER app
CMD ["leaderboard"]