#!/usr/bin/env python3

import logging
logging.basicConfig(level=logging.DEBUG)
import websocket
try:
    import thread
except ImportError:
    import _thread as thread
import sys
import time

def on_message(ws, message):
    print('MSG: ' + message)

def on_error(ws, error):
    print('ERROR: ' + error)

def on_close(ws):
    print("### closed ###")

def on_open(ws):
    def run(*args):
        for i in range(3):
            time.sleep(1)
            ws.send("Hello %d" % i)
        ws.close()
        print("thread terminating...")
    thread.start_new_thread(run, ())


if __name__ == "__main__":
    syslog_hostname = sys.argv[1]
    websocket.enableTrace(True)
    ws = websocket.WebSocketApp("ws://127.0.0.1:8080/logstream/%s" % syslog_hostname,
                              on_message = on_message,
                              on_error = on_error,
                              on_close = on_close)
    #ws.on_open = on_open
    ws.run_forever()
