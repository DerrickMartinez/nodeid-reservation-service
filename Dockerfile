FROM golang as builder

WORKDIR /go/src/nodeid-reservation-service
COPY . .
RUN make dep
RUN make build

# final image
FROM alpine:3.6
MAINTAINER Derrick Martinez <derrick.martinez@clearcapital.com>

RUN apk --no-cache add \
    ca-certificates

COPY --from=builder /go/src/nodeid-reservation-service/build/nodeid-reservation-service /bin/nodeid-reservation-service

EXPOSE 8080/tcp
ENTRYPOINT ["/bin/nodeid-reservation-service"]
