FROM golang:1.23-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/visit-gateway ./main.go

FROM alpine:3.20
RUN addgroup -S app && adduser -S app -G app
COPY --from=build /out/visit-gateway /visit-gateway
USER app
EXPOSE 8080
ENTRYPOINT ["/visit-gateway"]
