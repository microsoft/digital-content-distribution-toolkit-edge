import os

b2c_tenant = "mishtu"
b2c_app_tenant_id = os.environ["b2c_app_tenant_id"]
signupsignin_user_flow_phone_number = "B2C_1A_SignUpOrSignInWithPhone"
authority_template = "https://{tenant}.b2clogin.com/{tenant}.onmicrosoft.com/{user_flow}"
issuer_template = "https://{tenant}.b2clogin.com/{tenant_id}/v2.0/"

CLIENT_ID = os.environ["CLIENT_ID"] # Application (client) ID of app registration

PHONE_SIGNUPIN_AUTHORITY = authority_template.format(
    tenant=b2c_tenant, user_flow=signupsignin_user_flow_phone_number)

ISSUER = issuer_template.format(tenant = b2c_tenant, tenant_id = b2c_app_tenant_id)

REDIRECT_PATH = "/getAToken"  # Used for forming an absolute URL to your redirect URI.
                              # The absolute URL must match the redirect URI you set
                              # in the app's registration in the Azure portal.

# These are the scopes you've exposed in the web API app registration in the Azure portal
SCOPE = []  # Example with two exposed scopes: ["demo.read", "demo.write"]

SESSION_TYPE = "filesystem"  # Specifies the token cache should be stored in server-side session

HUB_CRM_URL = "https://mishtu-dashboard.azurewebsites.net/api/v1/hub_detail"

HUB_CRM_API_KEY = os.environ["HUB_CRM_API_KEY"]