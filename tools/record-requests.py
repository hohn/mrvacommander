# Make sure mvrvacommander is not running.  Then record via
# 
#       python3 record-requests.py
# 
# Start the mrvacommander and play back via
# 
#       sh request_<timestamp>.curl
# 
from http.server import BaseHTTPRequestHandler, HTTPServer
import logging
import os

class RequestHandler(BaseHTTPRequestHandler):
    def _set_response(self):
        self.send_response(200)
        self.send_header('Content-type', 'text/html')
        self.end_headers()

    def do_GET(self):
        logging.info("GET request,\nPath: %s\nHeaders:\n%s\n", str(self.path), str(self.headers))
        self._set_response()
        self.wfile.write("GET request for {}".format(self.path).encode('utf-8'))
        self.save_request("GET")

    def do_POST(self):
        content_length = int(self.headers['Content-Length'])
        post_data = self.rfile.read(content_length)
        logging.info("POST request,\nPath: %s\nHeaders:\n%s\n\nBody:\n%s\n",
                     str(self.path), str(self.headers), post_data.decode('utf-8')[:10])
        self._set_response()
        self.wfile.write("POST request for {}".format(self.path).encode('utf-8'))
        self.save_request("POST", post_data)

    def save_request(self, method, body=None):
        request_id = str(self.log_date_time_string()).replace(" ", "_").replace(":", "-").replace("/", "-")
        body_filename = f"request_body_{request_id}.blob"
        curl_filename = f"request_{request_id}.curl"

        with open(curl_filename, "w") as log_file:
            # Write the curl command
            url = f"http://localhost:{self.server.server_port}{self.path}"
            log_file.write(f"curl -X {method} '{url}'")

            # Add headers
            for key, value in self.headers.items():
                log_file.write(f" -H '{key}: {value}'")

            # Add body if present
            if body:
                with open(body_filename, "w") as body_file:
                    body_file.write(body.decode('utf-8'))
                log_file.write(f" --data-binary '@{body_filename}'")

            log_file.write("\n")

def run(server_class=HTTPServer, handler_class=RequestHandler, port=8080):
    logging.basicConfig(level=logging.INFO)
    server_address = ('', port)
    httpd = server_class(server_address, handler_class)
    logging.info('Starting httpd...\n')
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        pass
    httpd.server_close()
    logging.info('Stopping httpd...\n')

if __name__ == '__main__':
    run()
