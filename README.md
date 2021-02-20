# syslogstream
A syslog server that relays syslog messages to websockets


## Building

Requirements:
 * Go 1.14
 * make

Just run the following from the repository root:

```bash
make
```
 
## Running

After building, run the following from the repository root:

```bash
sudo ./syslog2websocket
```

The syslog server listens on UDP port 514 and the webserver listens on port 8080.

Websocket clients connect to syslog hostname-specific URIs to listen in on incoming logs:

`ws://127.0.0.1:8080/logstream/<hostname>`
