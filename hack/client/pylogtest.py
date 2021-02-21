#!/usr/bin/env python3

import logging
from logging.handlers import SysLogHandler, QueueHandler, QueueListener
from logging import Formatter
from queue import SimpleQueue
from syslog_rfc5424_formatter import RFC5424Formatter
from rfc5424logging import Rfc5424SysLogHandler, NILVALUE
import requests
import sys


logging.basicConfig(level=logging.DEBUG)
LOG = logging.getLogger(__name__)
#LOG.addHandler(QueueHandler(log_queue))

syslog_handler = Rfc5424SysLogHandler(('127.0.0.1', 514),
                                      hostname='task-foo')
#syslog_handler.setFormatter(RFC5424Formatter(msgid='36b1308f-e0e2-4d4a-ae98-284f51f39a8a'))
LOG.addHandler(syslog_handler)

LOG.info('this is an info message')
LOG.debug('this is a debug message')
LOG.info('this is another info message')
LOG.debug('and this is another debug message')
try:
    raise RuntimeError('foo')
except Exception as ex:
    LOG.exception(ex)
#queue_listener.stop()
