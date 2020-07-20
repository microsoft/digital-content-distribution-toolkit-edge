import requests
import json
import sys

#creds = 'Basic YmluZWxvZ2luOkRxVWtiT2dBeEVUK1ZEWFZCQVZmTURyWUF5ZHl5RXVF'

# Use argument variable to fill this
class reset_acr():
    def __init__(self, creds):
        self.creds = creds

    def delete_tag(self, image, digest):
        url = "https://binelogin.azurecr.io/v2/{0}/manifests/sha256:{1}".format(image, digest)
        payload = {}
        headers = {
        'Authorization': self.creds,
        'Accept': 'application/vnd.docker.distribution.manifest.v2+json'
        }
        response = requests.request("DELETE", url, headers=headers, data = payload)
        return response

    def list_tags(self, image):
        url = "https://binelogin.azurecr.io/v2/{0}/tags/list".format(image)
        payload = {}
        headers = {
        'Authorization': self.creds
        }

        response = requests.request("GET", url, headers=headers, data = payload)
        return json.loads(response.text)

    def get_digest_from(self, image, tag):
        # Docker-Content-Digest
        url = "https://binelogin.azurecr.io/v2/{0}/manifests/{1}".format(image, tag)
        payload = {}
        headers = {
        'Authorization': self.creds,
        'Accept': 'application/vnd.docker.distribution.manifest.v2+json'
        }
        response = requests.request("GET", url, headers=headers, data = payload)
        return response.headers['Docker-Content-Digest']

# acr.py $(arm_image_dev)[image_id] $(Build.SourceVersion)[tag] $(acr_credentials)[credentials]

image_id = sys.argv[1]
current_tag = sys.argv[2]
creds = sys.argv[3]

acr = reset_acr(creds)
cloud_res = acr.list_tags(image_id)

if 'tags' in cloud_res: 
    for tag in cloud_res['tags']:
        digest = acr.get_digest_from(image_id, tag)
        print(tag + ":" + digest)
        if tag != "latest" and tag != current_tag:
            print('Deleting: ' + tag)
            acr.delete_tag(image_id, digest)