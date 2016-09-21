package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/berrunder/go-tcard"
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

//const token = ""

// Bot structure
type Bot struct {
	api      *tgbotapi.BotAPI
	userNums map[int64]string
}

func main() {
	var token string
	var webhookAddr string
	var port string
	var remove bool
	flag.StringVar(&webhookAddr, "webhook", "", "Webhook base address (e.g. https://www.google.com:8443). If not set, getUpdates polling will be used, and old webhook will be removed.")
	flag.StringVar(&port, "port", "8443", "Port to listen for incoming connections (needed only for webhook)")
	flag.StringVar(&token, "token", "", "Bot token value (required)")
	flag.BoolVar(&remove, "removeHook", false, "Don't start bot, only remove webhook")

	flag.Parse()

	if token == "" {
		token = os.Getenv("TCARDBOT_TOKEN")
	}

	if token == "" {
		log.Fatal("Token value is required.")
	}

	bot, err := NewBot(token)
	if err != nil {
		log.Panic(err)
	}

	env := strings.TrimSpace(os.Getenv("GO_ENV"))
	if env != "production" {
		log.Println("Debug mode enabled")
		bot.api.Debug = true
	}

	log.Printf("Authorized on account %s", bot.api.Self.UserName)

	if remove {
		_, err = bot.api.RemoveWebhook()
		if err != nil {
			log.Fatal(err)
		}

		os.Exit(0)
	}

	bot.Listen(webhookAddr, port)
}

// NewBot creates new bot instance
func NewBot(token string) (*Bot, error) {
	bot, err := tgbotapi.NewBotAPI(token)

	if err != nil {
		return nil, err
	}

	return &Bot{
		api:      bot,
		userNums: make(map[int64]string),
	}, nil
}

// Listen for incoming messages.
// If webhookAddr is empty, webhook will be removed (if any) and getUpdates polling will be used. In this case port also can be empty
func (bot *Bot) Listen(webhookAddr string, port string) {
	var updates <-chan tgbotapi.Update
	var err error

	if webhookAddr != "" {
		_, err = bot.api.SetWebhook(tgbotapi.NewWebhook(webhookAddr + "/" + bot.api.Token))
		if err != nil {
			log.Fatal(err)
		}

		updates = bot.api.ListenForWebhook("/" + bot.api.Token)
		addr := "127.0.0.1:" + port
		go http.ListenAndServe(addr, nil)
	} else {
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60

		if _, err = bot.api.RemoveWebhook(); err != nil {
			log.Fatal(err)
		}

		updates, err = bot.api.GetUpdatesChan(u)
		if err != nil {
			log.Panic(err)
		}
	}

	bot.serveUpdatesChan(updates)
}

func (bot *Bot) serveUpdatesChan(updates <-chan tgbotapi.Update) {
	for update := range updates {
		if update.Message == nil {
			continue
		}

		command := update.Message.Command()

		if command == "start" || command == "help" {
			bot.handleHelpCommand(update.Message.Chat.ID)
			continue
		}

		if command == "check" && bot.handleCheckCommand(update.Message) {
			continue
		}

		if bot.handleMatch(update.Message.Chat.ID, update.Message.Text) {
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Я не распознал во входных данных номера карты. Он должен состоять из 19 цифр.")

		go bot.api.Send(msg)
	}
}

func (bot *Bot) handleHelpCommand(chatID int64) bool {
	messageText := "/check - Проверить баланс карты по номеру. Если не указать номер, будет запрошен баланс последней проверенной вами карты (если бот её помнит).\n" +
		"/help - Помощь в использовании бота\n\n" +
		"Также для проверки баланса карты можно просто написать её номер (неважно, вместе или с пробелами)"
	msg := tgbotapi.NewMessage(chatID, messageText)

	go bot.api.Send(msg)

	return true
}

func (bot *Bot) handleCheckCommand(message *tgbotapi.Message) bool {
	num := extractNumber(strings.TrimSpace(message.CommandArguments()))

	if num == "" {
		num = bot.userNums[message.Chat.ID]
	} else {
		bot.userNums[message.Chat.ID] = num
	}

	if num != "" {
		go bot.answerToNum(message.Chat.ID, num)

		return true
	}

	return false
}

func (bot *Bot) handleMatch(chatID int64, msg string) bool {
	num := extractNumber(msg)

	if num != "" {
		bot.userNums[chatID] = num

		go bot.answerToNum(chatID, num)

		return true
	}

	return false
}

func (bot *Bot) answerToNum(chatID int64, num string) (tgbotapi.Message, error) {
	answer := fetchAnswer(num)

	msg := tgbotapi.NewMessage(chatID, answer)
	msg.ParseMode = "Markdown"

	return bot.api.Send(msg)
}

func fetchAnswer(num string) string {
	card, err := tcard.Fetch(num, "")

	if err != nil {
		log.Print("Error fetching data", err)
		return "Произошла ошибка при получении данных по карте. Проверьте, правильно ли введен номер карты."
	}

	return fmt.Sprintf(
		"Данные карты *%s* на *%s*: \nсумма *%v*, дата окончания действия *%s*",
		card.PAN, time.Now().Format("02.01.2006"),
		float32(card.Sum)/100,
		card.EndDate)
}

func extractNumber(message string) string {
	r := regexp.MustCompile(`\b(\d{4})\s*(\d{5})\s*(\d{5})\s*(\d{5})\b`)

	matches := r.FindStringSubmatch(message)

	if matches != nil {
		return strings.Join(matches[1:], "")
	}

	return ""
}
