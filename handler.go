package main

import (
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/linebot"

	"github.com/labstack/echo"
)

type AppHandler struct{}

func (h *AppHandler) PushMessage(c echo.Context) error {
	botNames := []string{
		"CHIMPANZEE", "CRAB", "RABBIT", "HAMSTER",
	}

	for _, botName := range botNames {
		channelSecret := os.Getenv(botName + "_SECRET")
		channelAccessToken := os.Getenv(botName + "_ACCESS_TOKEN")
		groupID := os.Getenv(botName + "_GROUP_ID")

		if channelSecret == "" || channelAccessToken == "" || groupID == "" {
			continue
		}

		bot, err := linebot.New(channelSecret, channelAccessToken)
		if err != nil {
			return err
		}

		_, err = bot.PushMessage(groupID, linebot.NewTextMessage("テストテスト")).Do()
		if err != nil {
			return err
		}
	}

	return c.NoContent(http.StatusOK)
}

func (h *AppHandler) Webhook(c echo.Context) error {
	botName := c.Param("botName")

	channelSecret := os.Getenv(botName + "_SECRET")
	channelAccessToken := os.Getenv(botName + "_ACCESS_TOKEN")

	bot, err := linebot.New(channelSecret, channelAccessToken)
	if err != nil {
		return err
	}

	lineEvents, err := bot.ParseRequest(c.Request())
	if err != nil {
		return err
	}

	for _, lineEvent := range lineEvents {
		groupID := ""
		switch lineEvent.Source.Type {
		case linebot.EventSourceTypeGroup:
			groupID = lineEvent.Source.GroupID
		}

		print(groupID)
	}

	return c.NoContent(http.StatusOK)
}
