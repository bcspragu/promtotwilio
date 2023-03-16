FROM golang:1.20-alpine as build

RUN mkdir /user && \
    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
    echo 'nobody:x:65534:' > /user/group

WORKDIR /build
RUN apk add --update --no-cache ca-certificates git tzdata && update-ca-certificates

COPY ./go.mod ./go.sum ./
RUN go mod download && go mod verify

COPY ./main.go ./server.go ./server_test.go ./

RUN go test ./... && CGO_ENABLED=0 go build \
    -installsuffix 'static' \
    -o /promtotwilio .

FROM scratch

COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /user/group /user/passwd /etc/
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /promtotwilio /promtotwilio

EXPOSE 8080
USER nobody:nobody

ENTRYPOINT ["./promtotwilio"]
