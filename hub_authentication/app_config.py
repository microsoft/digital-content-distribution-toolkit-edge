import os

b2c_tenant = "binehub"
signin_user_flow = "B2C_1_SignIn"
signupsignin_user_flow = "B2C_1_SignUpIn"
editprofile_user_flow = "B2C_1_ProfileEdit"
resetpassword_user_flow = "B2C_1_PasswordReset"
signupsignin_user_flow_phone_number = "B2C_1A_SignUpOrSignInWithPhone"
authority_template = "https://{tenant}.b2clogin.com/{tenant}.onmicrosoft.com/{user_flow}"

CLIENT_ID = "f1538c13-fc9d-4c86-a012-4135b7032a07" # Application (client) ID of app registration

CLIENT_SECRET = ".S3-AZ~xc_d0ab.RP.7iDO~hHgLsOmt1.7" # Placeholder - for use ONLY during testing.
# In a production app, we recommend you use a more secure method of storing your secret,
# like Azure Key Vault. Or, use an environment variable as described in Flask's documentation:
# https://flask.palletsprojects.com/en/1.1.x/config/#configuring-from-environment-variables
# CLIENT_SECRET = os.getenv("CLIENT_SECRET")
# if not CLIENT_SECRET:
#     raise ValueError("Need to define CLIENT_SECRET environment variable")

SIGN_IN_AUTHORITY = authority_template.format(
    tenant=b2c_tenant, user_flow=signin_user_flow)
AUTHORITY = authority_template.format(
    tenant=b2c_tenant, user_flow=signupsignin_user_flow)
B2C_PROFILE_AUTHORITY = authority_template.format(
    tenant=b2c_tenant, user_flow=editprofile_user_flow)
B2C_RESET_PASSWORD_AUTHORITY = authority_template.format(
    tenant=b2c_tenant, user_flow=resetpassword_user_flow)
PHONE_SIGNUPIN_AUTHORITY = authority_template.format(
    tenant=b2c_tenant, user_flow=signupsignin_user_flow_phone_number)

REDIRECT_PATH = "/getAToken"  # Used for forming an absolute URL to your redirect URI.
                              # The absolute URL must match the redirect URI you set
                              # in the app's registration in the Azure portal.

# This is the API resource endpoint
ENDPOINT = '' # Application ID URI of app registration in Azure portal

# These are the scopes you've exposed in the web API app registration in the Azure portal
SCOPE = []  # Example with two exposed scopes: ["demo.read", "demo.write"]

SESSION_TYPE = "filesystem"  # Specifies the token cache should be stored in server-side session

HUB_CRM_URL = "https://hub-crm.southeastasia.cloudapp.azure.com:8080/api/v1/hub_detail"
HUB_CRM_API_KEY = "@sOGFjdgiwoXVxgALTg+n8h1L0weWPBue0vh"