import html
import json
from collections import deque
from http.server import BaseHTTPRequestHandler, HTTPServer

notifications = deque(maxlen=50)

class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers['Content-Length'])))
        notifications.appendleft(body)
        print(json.dumps(body, ensure_ascii=False), flush=True)
        self.send_response(200)
        self.end_headers()

    def do_GET(self):
        body = '<html><meta charset="utf-8"><meta http-equiv="refresh" content="10"><h1>Alerts</h1>'
        if not notifications:
            body += "<p>Алертов пока нету</p>"
        for notification in notifications:
            body += '<pre>' + html.escape(json.dumps(notification, ensure_ascii=False, indent=2)) + '</pre><hr>'
        self.send_response(200)
        self.send_header('Content-Type', 'text/html; charset=utf-8')
        self.end_headers()
        self.wfile.write(body.encode())

HTTPServer(('0.0.0.0', 8080), Handler).serve_forever()
