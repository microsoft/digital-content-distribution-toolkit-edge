import base64
import jwt
import json

# configuration, these can be seen in valid JWTs from Azure B2C:
valid_audiences = ['f1538c13-fc9d-4c86-a012-4135b7032a07'] # id of the application 
issuer = 'https://binehub.b2clogin.com/e8f701ab-e20c-4100-8e12-b82232a4ef56/v2.0/' # iss

'''
class InvalidAuthorizationToken(Exception):
    def __init__(self, details):
        super().__init__('Invalid authorization token: ' + details)

def get_kid(token):
    headers = jwt.get_unverified_header(token)
    if not headers:
        raise InvalidAuthorizationToken('missing headers')
    try:
        return headers['kid']
    except KeyError:
        raise InvalidAuthorizationToken('missing kid')


def get_jwk(kid):
    jwks = json.loads(open('./jwks.json', 'r').read())
    for jwk in jwks.get('keys'):
        if jwk.get('kid') == kid:
            return jwk
    raise InvalidAuthorizationToken('kid not recognized')


def get_public_key(token):
    return rsa_pem_from_jwk(get_jwk(get_kid(token)))
'''

def validate_jwt(jwt_to_validate):
    jwks = json.loads(open('./jwks.json', 'r').read())
    for jwk in jwks['keys']:
        public_key = jwt.algorithms.RSAAlgorithm.from_jwk(json.dumps(jwk))
        decoded = jwt.decode(jwt_to_validate, public_key, 
                                    verify=True,
                                    algorithms=['RS256'],
                                    audience=valid_audiences,
                                    issuer=issuer)
    '''
    public_key = get_public_key(jwt_to_validate)
    decoded = jwt.decode(jwt_to_validate,
                         public_key,
                         verify=True,
                         algorithms=['RS256'],
                         audience=valid_audiences,
                         issuer=issuer)
    '''
    # the JWT is validated
    print("***********************token is valid*********************")
    print(decoded)


'''
def ensure_bytes(key):
    if isinstance(key, str):
        key = key.encode('utf-8')
    return key


def decode_value(val):
    decoded = base64.urlsafe_b64decode(ensure_bytes(val) + b'==')
    return int.from_bytes(decoded, 'big')


def rsa_pem_from_jwk(jwk):
    return RSAPublicNumbers(
        n=decode_value(jwk['n']),
        e=decode_value(jwk['e'])
    ).public_key(default_backend()).public_bytes(
        encoding=serialization.Encoding.PEM,
        format=serialization.PublicFormat.SubjectPublicKeyInfo
    )
'''