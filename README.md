# echo
forward group notifications

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
