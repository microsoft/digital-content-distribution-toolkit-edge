from http.server import BaseHTTPRequestHandler, HTTPServer
import argparse
import re
import cgi
import json
import threading
from urllib import parse

PORT = 8134
contentID = {"6639"}
listcontents = {
                "status":"0",
                "listContents":
                [
                    {
                        "ContentId":"6639"
                    }
                ]
                }
metadata = {
  "status": "0",
  "metadata": {
    "editorID": "SES",
    "displayedName": "",
    "videoVersion": "SD",
    "keywords": "",
    "audioVersion": "",
    "productionDate": "0",
    "longSummary": "",
    "category": "test",
    "shortSummary": "ses-test-22mar21_1043",
    "productionNationality": "",
    "shortTitle": "ses-test-22mar21_1043",
    "longTitle": "",
    "actors": "",
    "comment": "",
    "directors": "",
    "duration": "PT00H17M01S",
    "series": "false",
    "userDefined": {
      "title": "MStore_tests"
    },
    "movieID": "22mar21_1043_16MB",
    "price": "0.00",
    "validityStartDate": "2021-03-22T09:07:56+00:00",
    "validityEndDate": "2021-03-31T18:24:56+00:00",
    "CID": "6639",
    "theoricDuration": "",
    "video filename": "/media/sda1/mstore/QCAST.ipts/storage/6639_22mar21_1043_16MB_210322090756_210331182456/22mar21_1043_vod_16MB.mp4",
    "trailer filename": "/media/sda1/mstore/QCAST.ipts/storage/6639_22mar21_1043_16MB_210322090756_210331182456/",
    "cover filename": "/media/sda1/mstore/QCAST.ipts/storage/6639_22mar21_1043_16MB_210322090756_210331182456/22mar21_1043_img2.jpg",
    "urlForDataFiles": "/media/sda1/mstore/QCAST.ipts/storage/6639_22mar21_1043_16MB_210322090756_210331182456/",
    "dataFiles": {
      "file": [
        {
          "filename": "22mar21_1043_bine_metadata.json",
          "filesize": 695
        },
        {
          "filename": "22mar21_1043_bine_metadata2.json",
          "filesize": 695
        }
      ]
    },
    "thumbnail filename": "/media/sda1/mstore/QCAST.ipts/storage/6639_22mar21_1043_16MB_210322090756_210331182456/22mar21_1043_img1.jpg"
  }
}
class HTTPRequestHandler(BaseHTTPRequestHandler):
    def set_headers(self):
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.end_headers()
    def do_GET(self):
        self.set_headers()
        if re.search('/listcontents', self.path):
            resp = json.dumps(listcontents)
            self.wfile.write(resp.encode('utf8'))
            return
        if re.search('/getmetadata/[0-9]+', self.path):
            cid = self.path.split('/')[-1]
            if cid in contentID: 
                resp = json.dumps(metadata)
                self.wfile.write(resp.encode('utf8'))
                return
            else:
                resp = json.dumps({"Resp": "Invalid request"})
                self.wfile.write(resp.encode('utf8'))
                return
        else:
            resp = json.dumps({"Resp": "Invalid request"})
            self.wfile.write(resp.encode('utf8'))
            return



def main():
    server = HTTPServer(('', PORT), HTTPRequestHandler)
    print('Mock HTTP Server Running...........')
    server.serve_forever()


if __name__ == '__main__':
    main()