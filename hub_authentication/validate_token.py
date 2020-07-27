import jwt
import json

# configuration, these can be seen in valid JWTs from Azure B2C:
app_id = 'f1538c13-fc9d-4c86-a012-4135b7032a07' # id of the application 
issuer = 'https://binehub.b2clogin.com/e8f701ab-e20c-4100-8e12-b82232a4ef56/v2.0/' # iss


def validate_jwt(jwt_to_validate):
    jwks = json.loads(open('./hub_authentication/jwks.json', 'r').read())
    for jwk in jwks['keys']:
        public_key = jwt.algorithms.RSAAlgorithm.from_jwk(json.dumps(jwk))
        jwt.decode(jwt_to_validate, public_key, 
                                    verify=True,
                                    algorithms=['RS256'],
                                    audience=app_id,
                                    issuer=issuer)
    # the JWT is validated
    print("Token is valid")
    #print(decoded)
