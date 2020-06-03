# echo

Send log to Mixin Messenger Groups.

# How to use Echo?

## Config Scanner

### 1. Download echo binary

- Download from https://github.com/fox-one/echo/releases.
- Extract and upload `scanner` to server. 

### 2. Get Token

- Create a new group in Mixin Messenger
- add echo bot to it
- copy the token.

### 3. Config echo

write a .env file

```
SCANNER_TOKEN_INFO=YOUR_TOKEN_TO_SEND_INFO_LOG
SCANNER_TOKEN_ERROR=YOUR_TOKEN_TO_SEND_ERROR_LOG
SCANNER_TOKEN_WARNING=YOUR_TOKEN_TO_SEND_WARNING_LOG
```

add echo as a service

```
[Unit]
Description=Scanner Service
After=network.target

[Service]
Type=simple
User=YOUR_NAME
Restart=on-failure
RestartSec=1s
EnvironmentFile=YOUR_ENV_FILE_PATH
ExecStart=/YOUR_ECHO_FILE_PATH/scanner --format post --cmd '["/bin/journalctl","-u","THE_SERVICE_YOU_WANNA_WATCH","-f","-o","cat"]'

[Install]
WantedBy=multi-user.target
```

### 4. Done

- use `github.com/sirupsen/logrus` and enable json log format in your service.
- restart all services

## Config Echo Server

### Blaze

接收入群消息，然后自动发 token 给拉机器人进群的人

### Server

```http request
POST https://echo.yiplee.com/message
```

**Header:**

Authorization: Bearer access_token

**Params:**

```json5
{
   "category": "Plain_Text",
   "data": "base64 msg data"
}
```


### Rate Limit

一分钟一百个请求
