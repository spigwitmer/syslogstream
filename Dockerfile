FROM golang:1.16.0

RUN mkdir -p /build /app
COPY cmd/ /build/cmd
COPY go.sum go.mod /build/
RUN cd /build && go build -ldflags="-extldflags=-static" ./cmd/syslog2websocket
RUN cp /build/syslog2websocket /app/

EXPOSE 514/udp
EXPOSE 8080/tcp

CMD ["/app/syslog2websocket"]
