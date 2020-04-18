package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"

	"github.com/google/go-jsonnet"
	"github.com/line/line-bot-sdk-go/linebot"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/labstack/echo"
)

type AppHandler struct {
	problem *Problem
	answers map[string]*Answer
}

type Problem struct {
	ID           int
	Text         string
	ImageURL     string
	Setter       string
	Difficulty   int
	Options      []string
	Editorial    string
	Comment      string
	HasSubmitted bool
}

func (p *Problem) FromRow(header []interface{}, row []interface{}) bool {
	options := []string{}
	for index, value := range row {
		switch header[index] {
		case "番号":
			id, _ := strconv.Atoi(value.(string))
			p.ID = id
		case "画像ID":
			imageID := value.(string)
			if imageID != "" {
				p.ImageURL = "https://drive.google.com/uc?export=view&id=" + imageID
			}
		case "出題文":
			p.Text = value.(string)
		case "出題者":
			p.Setter = value.(string)
		case "難易度":
			difficulty, _ := strconv.Atoi(value.(string))
			p.Difficulty = difficulty
		case "選択肢1":
			if option := value.(string); option != "" {
				options = append(options, option)
			}
		case "選択肢2":
			if option := value.(string); option != "" {
				options = append(options, option)
			}
		case "選択肢3":
			if option := value.(string); option != "" {
				options = append(options, option)
			}
		case "選択肢4":
			if option := value.(string); option != "" {
				options = append(options, option)
			}
		case "解説":
			p.Editorial = value.(string)
		case "備考":
			p.Comment = value.(string)
		case "出題済":
			hasSubmitted, _ := strconv.Atoi(value.(string))
			p.HasSubmitted = (hasSubmitted == 1)
		}
	}

	p.Options = options
	return p.ImageURL != ""
}

func (h *AppHandler) readProblems(ctx context.Context) ([]*Problem, error) {
	b, err := ioutil.ReadFile(GoogleCredentialPath())
	if err != nil {
		return []*Problem{}, err
	}

	credential := map[string]interface{}{}
	err = json.Unmarshal(b, &credential)
	if err != nil {
		return []*Problem{}, err
	}

	config := &jwt.Config{
		Email:      credential["client_email"].(string),
		PrivateKey: []byte(credential["private_key"].(string)),
		Scopes: []string{
			"https://www.googleapis.com/auth/drive",
		},
		TokenURL: google.JWTTokenURL,
	}

	sheetService, err := sheets.NewService(ctx, option.WithTokenSource(config.TokenSource(oauth2.NoContext)))
	if err != nil {
		return []*Problem{}, err
	}

	valueRange, err := sheetService.Spreadsheets.Values.Get(ProblemSheetID(), "問題!A1:K1000").Do()
	if err != nil {
		return []*Problem{}, err
	}

	header := valueRange.Values[0]
	problems := []*Problem{}
	for index, row := range valueRange.Values {
		if index == 0 {
			continue
		}

		problem := new(Problem)
		if problem.FromRow(header, row) {
			problems = append(problems, problem)
		}
	}

	return problems, nil
}

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
	if h.problem == nil {
		return c.NoContent(http.StatusOK)
	}

	return c.Render(http.StatusOK, "liff.html", map[string]interface{}{
		"options": h.problem.Options,
	})
}

func (h *AppHandler) LiffSubmit(c echo.Context) error {
	if h.problem == nil {
		return c.NoContent(http.StatusOK)
	}

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

func (h *AppHandler) PushProblem(ctx context.Context) error {
	problems, err := h.readProblems(ctx)
	if err != nil {
		return err
	}

	if len(problems) == 0 {
		return nil
	}

	h.problem = problems[rand.Intn(len(problems))]
	h.answers = map[string]*Answer{}

	botNames := []string{
		"CHIMPANZEE", "CRAB", "RABBIT", "HAMSTER",
	}

	for _, botName := range botNames {
		channelAccessToken := ChannelAccessToken(botName)
		groupID := GroupID(botName)

		if channelAccessToken == "" || groupID == "" {
			continue
		}

		h.pushFlexMessage(
			ctx,
			channelAccessToken,
			groupID,
			"今日の1レッグ",
			ProblemTemplatePath(),
			map[string]interface{}{
				"imageURL":   h.problem.ImageURL,
				"text":       h.problem.Text,
				"difficulty": h.problem.Difficulty,
				"setter":     h.problem.Setter,
			},
		)
	}

	return nil
}
