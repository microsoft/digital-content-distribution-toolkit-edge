import os

b2c_tenant = "mishtu"
signupsignin_user_flow_phone_number = "B2C_1A_SignUpOrSignInWithPhone"
authority_template = "https://{tenant}.b2clogin.com/{tenant}.onmicrosoft.com/{user_flow}"

CLIENT_ID = "439695b8-dff9-43dd-af0e-fc9921a7ccca" # Application (client) ID of app registration

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

HUB_CRM_URL = "https://hub-management.azurewebsites.net/api/v1/hub_detail"
HUB_CRM_API_KEY = "@sOGFjdgiwoXVxgALTg+n8h1L0weWPBue0vh"