package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/google/go-jsonnet"
	"github.com/line/line-bot-sdk-go/linebot"

	"github.com/labstack/echo"
)

type AppHandler struct{}

func (h *AppHandler) pushFlexMessage(ctx context.Context, accessToken string, to string, altText string, templateFilePath string, args map[string]interface{}) error {
	argsJson, _ := json.Marshal(args)

	file, err := os.Open(templateFilePath)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	vm := jsonnet.MakeVM()
	vm.ExtVar("args", string(argsJson))
	flexJson, err := vm.EvaluateSnippet(templateFilePath, string(b))
	if err != nil {
		return err
	}

	body := fmt.Sprintf(`{
		"to": "%s",
		"messages": [{
            "type": "flex",
            "altText": "%s",
            "contents": %s
		}]
	}`, to, altText, flexJson)

	request, err := http.NewRequest(
		"POST",
		"https://api.line.me/v2/bot/message/push",
		bytes.NewBuffer([]byte(body)),
	)

	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+accessToken)

	_, err = (&http.Client{}).Do(request)
	return err
}

func (h *AppHandler) PushMessage(c echo.Context) error {
	ctx := c.Request().Context()
	botNames := []string{
		"CHIMPANZEE", "CRAB", "RABBIT", "HAMSTER",
	}

	for _, botName := range botNames {
		channelAccessToken := ChannelAccessToken(botName)
		groupID := GroupID(botName)

		if channelAccessToken == "" || groupID == "" {
			continue
		}

		err := h.pushFlexMessage(
			ctx,
			channelAccessToken,
			groupID,
			"今日の1レッグ",
			ProblemTemplatePath(),
			map[string]interface{}{
				"imageURL":   "https://drive.google.com/uc?export=view&id=1NbJIsKqta98RttbCCQJ0_ajjCFPDRKPW",
				"text":       "2019年度モデルレース2→3",
				"difficulty": 4,
				"setter":     "岩井",
				"options": []string{
					"岩崖つき水系に当てて左を登っていく",
					"まっすぐ気味に道を繋ぎながら進む",
					"右の舗装路まで出て上から突撃",
				},
			},
		)

		if err != nil {
			return err
		}
	}

	return c.NoContent(http.StatusOK)
}

func (h *AppHandler) Webhook(c echo.Context) error {
	botName := c.Param("botName")

	channelSecret := ChannelSecret(botName)
	channelAccessToken := ChannelAccessToken(botName)

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

		print(groupID + "\n")
	}

	return c.NoContent(http.StatusOK)
}

func (h *AppHandler) LiffPage(c echo.Context) error {
	return c.Render(http.StatusOK, "liff.html", map[string]interface{}{
		"options": []string{"選択肢1", "選択肢2", "選択肢3", "選択肢4"},
	})
}

func (h *AppHandler) LiffSubmit(c echo.Context) error {
	type Parameter struct {
		UserID   string `json:"userID"`
		UserName string `json:"userName"`
		Option   int    `json:"option"`
		Comment  string `json:"comment"`
	}

	param := new(Parameter)
	if err := c.Bind(param); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid parameter")
	}

	print(param.UserID)
	print(param.UserName)
	print(param.Option)
	print(param.Comment)
	return c.NoContent(http.StatusOK)
}
