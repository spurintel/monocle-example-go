# monocle-example-go
A simple Go backend example to get you started with monocle. It includes a Go web server and some basic HTML with a login form protected by monocle.

## What is monocle?
Monocle is a passive zero-trust captcha that provides your web application with data to assess the risk of an individual user connection. Monocle is the only tool of its class capable of detecting residential proxies.

## How does monocle work?
You add a small JavaScript stub to your website or application. On a user-action, such as a form submission, you get an assessment (a.k.a threat bundle) that you can interpret on your backend to take action.

## Getting started

### Sign up for monocle
Before you can use this example you need a monocle private key and token.

1. Create a free Spur account - [Spur Sign Up](https://spur.us/app/start/create-account)
2. Sign in to your account
3. Navigate to your monocle settings - [Monocle Management](https://spur.us/app/monocle)
4. Create a deployment
5. Save your deployment key and site token
6. Setup your private key for embedding as an env variable. (Mac OS example)
    ```
    cat monocle-key.pem|base64|pbcopy
    ```
7. Use the base64 encoded key in your .env file.

### Dependencies
You need to have Go, Docker, and make installed

### Environment
For local testing you need to setup an env file.

Setup an environment file called .env in this directory. It should look like the following:
```
PORT=8080
PRIVATE_KEY={YOUR_PRIVATE_KEY_HERE}
TOKEN={YOUR_TOKEN}
USERNAME=alice
PASSWORD=alice
```

You can change the username and password to anything you want. It is only for testing purposes.

## Running
### Local
You can run the server locally by executing `make run`. This will build a binary based on your local system architecture and start the server.

Navigate to http://localhost:8080

#### Form/Landing Page

![Form Image](images/form.png)

#### Success Page

![Success Page Image](images/success.png)

#### Unauthorized Page

![Unauthorized Page Image](images/unauthorized.png)

### Building a container

### Heroku
