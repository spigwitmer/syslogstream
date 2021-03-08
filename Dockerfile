FROM alpine:3

RUN mkdir -p /app
COPY syslog2websocket /app/
RUN chmod +x /app/syslog2websocket

CMD ["/app/syslog2websocket"]
