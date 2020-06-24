from flask import Flask, render_template, redirect, url_for, request
import configparser

app = Flask(__name__)

config = configparser.ConfigParser()

@app.route('/welcome')
def welcome():
    return render_template('welcome.html')

@app.route('/home')
def home():
    return render_template('home.html')

@app.route('/login', methods=['GET', 'POST'])
def start():
    error = None
    if request.method == 'POST':
        if len(request.form['name']) == 0 or len(request.form['location']) == 0 or len(request.form['phonenumber']) == 0 :
            error = 'Invalid Name or Store location or Phone Number'
        else:
            name = request.form['name']
            location = request.form['location']
            phonenumber = request.form['phonenumber']
            hubDetails = open('../device_sdk/hubdetails.ini', 'w+') 
            name_config = f'customer_name={name}\n'
            location_config = f'location={location}\n'
            phonenumber_config = f'phonenumber={phonenumber}\n'
            hubDetails.write("[customer_details]\n") 
            hubDetails.write(name_config)
            hubDetails.write(location_config)
            hubDetails.write(phonenumber_config)
            hubDetails.close() 
            error = None
            return redirect(url_for('home'))
    return render_template('login.html', error=error)

@app.route('/<path:dummy>')
def fallback(dummy):
    return redirect(url_for('welcome'))

if __name__ == '__main__':
    config.read('../hub_config.ini')
    print(config.sections())
    app.run(debug=True, host="0.0.0.0", port=config.getint("HUB_AUTHENTICATION", "FLASK_PORT"))
