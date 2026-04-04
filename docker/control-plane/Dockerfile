FROM golang:1.26.1-alpine AS build

WORKDIR /src

COPY go.mod go.work ./
COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/control-plane ./cmd/control-plane

FROM alpine:3.21

RUN adduser -D -u 10001 safe

USER safe
WORKDIR /app

COPY --from=build /out/control-plane /app/control-plane

EXPOSE 8080

ENTRYPOINT ["/app/control-plane"]
