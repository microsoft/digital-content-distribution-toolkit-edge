import configparser
from flask import Flask, render_template, session, request, redirect, url_for
from flask_session import Session  
import msal
import app_config
import uuid
import requests
import subprocess
import os
from validate_token import validate_jwt

app = Flask(__name__)
app.config.from_object(app_config)
Session(app)

config = configparser.ConfigParser()

@app.route("/")
def home():
    if not session.get("user"):
        return redirect(url_for("login"))
    return render_template('home.html', user=session["user"])

@app.route("/login")
def login():
    session["state"] = str(uuid.uuid4())
    auth_url = _build_auth_url(scopes=app_config.SCOPE, state=session["state"])
    return render_template("login.html", auth_url=auth_url)

@app.route(app_config.REDIRECT_PATH)  # Its absolute URL must match your app's redirect_uri set in AAD
def authorized():
    if "error" in request.args:  # Authentication/Authorization failure
        print("error : Authentication/Authorization failure")
        return render_template("auth_error.html", result=request.args)
    if request.args.get('code'):
        cache = _load_cache()
        result = _build_msal_app(cache=cache).acquire_token_by_authorization_code(
            request.args['code'],
            scopes=app_config.SCOPE,  # Misspelled scope would cause an HTTP 400 error here
            redirect_uri=url_for("authorized", _external=True))
        if "error" in result:
            return render_template("auth_error.html", result=result)
        
        jwt = result.get("id_token")
        try:
            validate_jwt(jwt)
        except Exception as e:
            print("Exception while validating token caught. Error: {}".format(e))
        
        session["user"] = result.get("id_token_claims")
        _save_cache(cache)
    return render_template('register.html', user=session["user"])

@app.route('/register', methods=['POST'])
def start():
    error = None
    if request.method == 'POST':
        if len(request.form['storename']) == 0 or len(request.form['storelocation']) == 0 :
            error = 'Invalid Store Name or Store location'
        else:
            # save store and user details in customerdetails.ini file
            customer_name = session["user"].get("name")
            customer_contact = session["user"].get("signInNames.phoneNumber") # extension_Contact
            device_name = request.form['devicename']
            store_name = request.form['storename']
            store_location = request.form['storelocation']
            customerDetails = open('customerdetails.ini', 'w') 
            customer_name_config = f'customer_name={customer_name}\n'
            customer_contact_config = f'customer_contact={customer_contact}\n'
            device_name_config = f'device_name={device_name}\n'
            store_name_config = f'store_name={store_name}\n'
            store_location_config = f'store_location={store_location}\n'
            customerDetails.write("[customer_details]\n") 
            customerDetails.write(customer_name_config)
            customerDetails.write(customer_contact_config)
            customerDetails.write(device_name_config)
            customerDetails.write(store_name_config)
            customerDetails.write(store_location_config)
            customerDetails.close() 
            
            #submit the device details and register the device with the CRM application
            config.read('device.ini') 
            payload = {
                "apiKey":  app_config.HUB_CRM_API_KEY,
                "device_id": config.get('section_device','deviceId'),
                "device_name": device_name,
                "shop_name": store_name,
                "contact": customer_contact
            }
            requests.post(url = app_config.HUB_CRM_URL, data = payload)
            
            #Create dummy file in tmp directory
            os.mkdir("tmp")
            f = open("./tmp/dummy","w")
            f.close
            error = None
            
            # run the start_hub.sh script
            subprocess.run(['./start_hub.sh'])
            
            return render_template('home.html', user=session["user"], deviceName = device_name, storeName = store_name, location = store_location )
    return render_template('register.html', error=error)

@app.route('/<path:dummy>')
def fallback(dummy):
    return redirect(url_for('home'))

def _build_msal_app(cache=None, authority=None):
    return msal.ConfidentialClientApplication(
        app_config.CLIENT_ID, authority=authority or app_config.PHONE_SIGNUPIN_AUTHORITY,
        client_credential=app_config.CLIENT_SECRET, token_cache=cache)

def _build_auth_url(authority=None, scopes=None, state=None):
    return _build_msal_app(authority=authority).get_authorization_request_url(
        scopes or [],
        state=state or str(uuid.uuid4()),
        nonce="defaultNonce",
        prompt="login",
        redirect_uri=url_for("authorized", _external=True))

def _load_cache():
    cache = msal.SerializableTokenCache()
    if session.get("token_cache"):
        cache.deserialize(session["token_cache"])
    return cache

def _save_cache(cache):
    if cache.has_state_changed:
        session["token_cache"] = cache.serialize()
        
if __name__ == '__main__':
    config.read('hub_config.ini')
    print(config.sections())
    app.run(debug=True, host="0.0.0.0", port=config.getint("HUB_AUTHENTICATION", "FLASK_PORT"))
