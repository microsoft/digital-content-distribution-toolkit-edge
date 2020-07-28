import jwt
import json
import app_config


def validate_jwt(jwt_to_validate):
    jwks = json.loads(open('./hub_authentication/jwks.json', 'r').read())
    for jwk in jwks['keys']:
        public_key = jwt.algorithms.RSAAlgorithm.from_jwk(json.dumps(jwk))
        jwt.decode(jwt_to_validate, public_key, 
                                    verify=True,
                                    algorithms=['RS256'],
                                    audience=app_config.CLIENT_ID,
                                    issuer=app_config.ISSUER)
    # the JWT is validated
    print("Token is valid")
    #print(decoded)
