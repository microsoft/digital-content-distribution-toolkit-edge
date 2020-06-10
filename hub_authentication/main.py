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
        if request.form['password'] != config.get("HUB_AUTHENTICATION", "HUB_TOKEN"):
            error = 'Invalid Credentials. Please try again.'
        else:
            return redirect(url_for('home'))
    return render_template('login.html', error=error)

@app.route('/<path:dummy>')
def fallback(dummy):
    return redirect(url_for('welcome'))

if __name__ == '__main__':
    config.read('../hub_config.ini')
    print(config.sections())
    app.run(debug=True, host="0.0.0.0", port=config.getint("HUB_AUTHENTICATION", "FLASK_PORT"))
