FROM golang:1.13 as build

WORKDIR /go/src/app
COPY /src .
RUN go get
RUN CGO_ENABLED=0 go build -o /paxful

##
## Deploy
##
FROM alpine

WORKDIR /
COPY --from=build /paxful /paxful

EXPOSE 8080

ENTRYPOINT ["/paxful"]
