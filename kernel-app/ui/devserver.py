# Превью интерфейса в браузере (только для разработки, в .exe не входит).
# Отдаёт app.html с подключённым mock.js вместо Go-биндингов.
# Запуск: python kernel-app/ui/devserver.py  →  http://127.0.0.1:8765/
import http.server, os, sys

UI = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.dirname(UI)
PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 8765

TYPES = {".html": "text/html; charset=utf-8", ".js": "application/javascript; charset=utf-8", ".otf": "font/otf"}

class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        p = self.path.split("?")[0].lstrip("/")
        if p in ("", "index.html"):
            with open(os.path.join(UI, "app.html"), encoding="utf-8") as f:
                html = f.read().replace('<script src="app.js"></script>', '<script src="mock.js"></script>\n<script src="app.js"></script>')
            return self.send(html.encode("utf-8"), TYPES[".html"])
        path = os.path.join(ROOT, p) if p.startswith("fonts/") else os.path.join(UI, p)
        if os.path.isfile(path):
            with open(path, "rb") as f:
                return self.send(f.read(), TYPES.get(os.path.splitext(path)[1], "application/octet-stream"))
        self.send_error(404)

    def send(self, body, ctype):
        self.send_response(200)
        self.send_header("Content-Type", ctype)
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *a):
        pass

http.server.ThreadingHTTPServer(("127.0.0.1", PORT), H).serve_forever()
