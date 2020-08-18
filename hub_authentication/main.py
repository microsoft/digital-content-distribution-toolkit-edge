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
from key_vault import retrieve_client_secret

app = Flask(__name__)
app.config.from_object(app_config)
Session(app)

config = configparser.ConfigParser()

@app.route("/")
def home():
    if session.get("user") and session.get("deviceDetails"):
        #return redirect(url_for("login"))
        return render_template('home.html', user=session["user"], deviceDetails = session["deviceDetails"])
    return redirect(url_for("login"))

@app.route('/<path:dummy>')
def fallback(dummy):
    return redirect(url_for('home'))

@app.route("/login")
def login():
    if session.get("user") and session.get("deviceDetails"):
        return redirect(url_for("home"))
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
        
        print(result)
        jwt = result.get("id_token")
        try:
            validate_jwt(jwt)
        except Exception as e:
            print("Exception while validating token caught. Error: {}".format(e))
        
        session["user"] = result.get("id_token_claims")
        _save_cache(cache)
        
        # if the retailer_detail.ini file is already present, indicates that the retailer has registered.
        # Fetch the existing details from the file on the welcome page
        if os.path.isfile(config.get('HUB_AUTHENTICATION','RETAILER_DETAIL_FILE')):
            config.read(config.get('HUB_AUTHENTICATION','DEVICE_DETAIL_FILE')) 
            device_id_info = config.get('DEVICE_DETAIL','deviceId')
            # get and store device details to session from the retailer ini file
            retailer_file = open(config.get('HUB_AUTHENTICATION','RETAILER_DETAIL_FILE'), "r")
            file_lines = retailer_file.readlines()
            for line in file_lines:
                index = line.find("=")
                if index != -1:
                    if line.find("device_name") != -1:
                        device_name_info = line[index+1:]
                    elif line.find("store_name") != -1:
                        store_name_info = line[index+1:]
                    elif line.find("store_location") != -1:
                        store_location_info = line[index+1:]
                        
            deviceDetails = {
                "device_name": device_name_info,
                "store_name": store_name_info,
                "store_location": store_location_info,
                "device_id": device_id_info
            }
            session["deviceDetails"] = deviceDetails
            # run the start_hub.sh script
            subprocess.run(['./run_go_python_sides.sh'])
            return redirect(url_for('home'))
        
    return render_template('register.html', user=session["user"])

@app.route('/register', methods=['GET','POST'])
def start():
    if session.get("user") and session.get("deviceDetails"):
        return redirect(url_for("home"))
    if not session.get("user"):
        return redirect(url_for("login"))
    error = None
    
    if request.method == 'POST':
        if len(request.form['storename']) == 0 or len(request.form['storelocation']) == 0 or len(request.form['devicename']) == 0 :
            error = 'Invalid Device name or Store Name or Store location'
        else:
            # save store and user details in retailerdetails.ini file
            retailer_name = session["user"].get("name")
            retailer_contact = session["user"].get("signInNames.phoneNumber") # extension_Contact
            device_name = request.form['devicename']
            store_name = request.form['storename']
            store_location = request.form['storelocation']
            
            retailer_name_config = f'retailer_name={retailer_name}\n'
            device_name_config = f'device_name={device_name}\n'
            store_name_config = f'store_name={store_name}\n'
            store_location_config = f'store_location={store_location}\n'
            
            # Create retailer details file in path given in hub_config.ini file
            # the directory has to be present as device_details is also stored in same directory
            retailer_detail_file = config.get('HUB_AUTHENTICATION','RETAILER_DETAIL_FILE')
            retailerDetails = open(retailer_detail_file,"w")
            retailerDetails.write("[RETAILER_DETAIL]\n") 
            retailerDetails.write(retailer_name_config)
            retailerDetails.write(device_name_config)
            retailerDetails.write(store_name_config)
            retailerDetails.write(store_location_config)
            retailerDetails.close() 
            
            #submit the device details and register the device with the CRM application
            config.read(config.get('HUB_AUTHENTICATION','DEVICE_DETAIL_FILE')) 
            device_id = config.get('DEVICE_DETAIL','deviceId')
            payload = {
                "api_key":app_config.HUB_CRM_API_KEY,
                "deviceID": device_id,
                "name":device_name,
                "shop_name":store_name,
                "contact":retailer_contact
            }
            response = requests.post(url = app_config.HUB_CRM_URL, data = payload)
            print(response.content)
            print(response.status_code)
            if not response.status_code == 201: 
                return render_template('register.html', error=response.content, user=session["user"])
            

            #store device details to session
            deviceDetails = {
               "device_name": device_name,
               "store_name": store_name,
               "store_location": store_location,
               "device_id": device_id
            }
            session["deviceDetails"] = deviceDetails
            error = None
            
            # run the start_hub.sh script
            subprocess.run(['./run_go_python_sides.sh'])
            
            return redirect(url_for('home'))
    return render_template('register.html', error=error, user=session["user"])

          
@app.route("/logout")
def logout():
    session.clear()  # Wipe out user and its token cache from session
    # delete the retailer details, if exists
    if os.path.isfile(config.get('HUB_AUTHENTICATION','RETAILER_DETAIL_FILE')):
        os.remove(config.get('HUB_AUTHENTICATION','RETAILER_DETAIL_FILE'))
    return redirect(  # Also logout from your tenant's web session
        app_config.PHONE_SIGNUPIN_AUTHORITY + "/oauth2/v2.0/logout" +
        "?post_logout_redirect_uri=" + url_for("home", _external=True))

def _build_msal_app(cache=None, authority=None):
    return msal.ConfidentialClientApplication(
        app_config.CLIENT_ID, authority=authority or app_config.PHONE_SIGNUPIN_AUTHORITY,
        client_credential=retrieve_client_secret(), token_cache=cache)

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
    app.run(debug=False, host="0.0.0.0", port=config.getint("HUB_AUTHENTICATION", "FLASK_PORT"), ssl_context=('mishtu.crt', 'mishtu.key'))
