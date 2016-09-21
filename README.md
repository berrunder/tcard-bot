# TCard bot

Simple telegram bot to check balance of transport cards used in Krasnodar, Russia. Bot is unofficial, it simply parses data from https://t-karta.ru.
Bot can be found at http://telegram.me/TcardBot.

### Usage
If you want to launch your own instance of bot, just compile it (`go build`) and run
```
bot -token="YourTelegramToken" [-webhook="http://webhookbaseurl"] [-port=8443]
```  

### Command line arguments
``` 
  -port string
        Port to listen for incoming connections (needed only for webhook) (default "8443")
  -removeHook
        Don't start bot, only remove webhook
  -token string
        Bot token value (required)
  -webhook string
        Webhook base address (e.g. https://www.google.com:8443). If not set, getUpdates polling will be used, and old webhook will be removed.
```