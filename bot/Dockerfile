FROM golang:1.26-alpine AS build
WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /lingw ./cmd/lingw

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /lingw /usr/local/bin/lingw
ENTRYPOINT ["/usr/local/bin/lingw"]
