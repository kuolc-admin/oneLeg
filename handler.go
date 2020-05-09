package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/google/go-jsonnet"
	"github.com/kuolc/oneLeg/consts"
	"github.com/kuolc/oneLeg/firebase_"
	"github.com/kuolc/oneLeg/json_"
	"github.com/line/line-bot-sdk-go/linebot"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/labstack/echo"
)

type ProblemID = string
type UserID = string

type AppHandler struct {
	todayProblem *Problem
	problems     map[ProblemID]*Problem
	answers      map[UserID]*Answer
	maps         []*OMap
}

type Problem struct {
	ID                string   `json:"-"`
	Index             int      `json:"index"`
	Text              string   `json:"text"`
	OriginalImageURL  string   `json:"originalImageURL"`
	ProblemImageURL   string   `json:"problemImageURL"`
	EditorialImageURL string   `json:"editorialImageURL"`
	Setter            string   `json:"setter"`
	Difficulty        int      `json:"difficulty"`
	Options           []string `json:"options"`
	Editorial         string   `json:"editorial"`
	Note              string   `json:"note"`
	HasSubmitted      bool     `json:"-"`
}

type Answer struct {
	ID           string `json:"-"`
	ProblemID    string `json:"problemID"`
	UserID       string `json:"userID"`
	UserName     string `json:"userName"`
	UserGroupID  string `json:"userGroupID"`
	UserIsHidden bool   `json:"userIsHidden"`
	Option       int    `json:"option"`
	Comment      string `json:"comment"`
}

type User struct {
	ID       string `json:"-"`
	Name     string `json:"name"`
	GroupID  string `json:"groupID"`
	IsHidden bool   `json:"isHidden"`
}

type OMap struct {
	Name       string   `json:"name"`
	Year       int      `json:"year"`
	Event      string   `json:"event"`
	Regulation string   `json:"regulation"`
	URLs       []string `json:"urls"`
}

func (p *Problem) FromRow(header []interface{}, row []interface{}) bool {
	options := []string{}
	for index, value := range row {
		switch header[index] {
		case "番号":
			index, _ := strconv.Atoi(value.(string))
			p.Index = index
		case "元画像ID":
			imageID := value.(string)
			if imageID != "" {
				p.OriginalImageURL = "https://drive.google.com/uc?export=view&id=" + imageID
			}
		case "出題画像ID":
			imageID := value.(string)
			if imageID != "" {
				p.ProblemImageURL = "https://drive.google.com/uc?export=view&id=" + imageID
			}
		case "出題文":
			p.Text = value.(string)
		case "出題者":
			p.Setter = value.(string)
		case "難易度":
			difficulty, _ := strconv.Atoi(value.(string))
			p.Difficulty = difficulty
		case "選択肢1", "選択肢2", "選択肢3", "選択肢4":
			if option := value.(string); option != "" {
				options = append(options, option)
			}
		case "解説画像ID":
			imageID := value.(string)
			if imageID != "" {
				p.EditorialImageURL = "https://drive.google.com/uc?export=view&id=" + imageID
			}
		case "解説文":
			p.Editorial = value.(string)
		case "備考":
			p.Note = value.(string)
		case "出題済":
			hasSubmitted, _ := value.(string)
			p.HasSubmitted = (hasSubmitted == "1")
		}
	}

	if p.ProblemImageURL == "" {
		p.ProblemImageURL = p.OriginalImageURL
	}

	p.Options = options
	return p.OriginalImageURL != ""
}

func (m *OMap) FromRow(header []interface{}, row []interface{}) bool {
	urls := []string{}
	for index, value := range row {
		switch header[index] {
		case "テレイン名":
			m.Name = value.(string)
		case "年度":
			year, _ := strconv.Atoi(value.(string))
			m.Year = year
		case "イベント名":
			m.Event = value.(string)
		case "競技形式":
			m.Regulation = value.(string)
		case "URL1", "URL2":
			if url := value.(string); url != "" {
				urls = append(urls, url)
			}
		}
	}

	m.URLs = urls
	return m.Name != ""
}

func (h *AppHandler) readProblems(ctx context.Context) ([]*Problem, error) {
	b, err := ioutil.ReadFile(consts.GoogleCredentialPath())
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

	valueRange, err := sheetService.Spreadsheets.Values.Get(consts.SheetID(), "問題!A1:N1000").Do()
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
		if problem.FromRow(header, row) && !problem.HasSubmitted {
			problems = append(problems, problem)
		}
	}

	return problems, nil
}

func (h *AppHandler) setProblemSubmitted(ctx context.Context, index int) error {
	b, err := ioutil.ReadFile(consts.GoogleCredentialPath())
	if err != nil {
		return err
	}

	credential := map[string]interface{}{}
	err = json.Unmarshal(b, &credential)
	if err != nil {
		return err
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
		return err
	}

	_, err = sheetService.Spreadsheets.Values.Update(consts.SheetID(), fmt.Sprintf("問題!N%d", index+1), &sheets.ValueRange{
		Values: [][]interface{}{
			[]interface{}{
				"1",
			},
		},
	}).ValueInputOption("USER_ENTERED").Do()

	return err
}

func (h *AppHandler) readOMaps(ctx context.Context) ([]*OMap, error) {
	b, err := ioutil.ReadFile(consts.GoogleCredentialPath())
	if err != nil {
		return []*OMap{}, err
	}

	credential := map[string]interface{}{}
	err = json.Unmarshal(b, &credential)
	if err != nil {
		return []*OMap{}, err
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
		return []*OMap{}, err
	}

	valueRange, err := sheetService.Spreadsheets.Values.Get(consts.SheetID(), "地図!A1:F1000").Do()
	if err != nil {
		return []*OMap{}, err
	}

	header := valueRange.Values[0]
	maps := []*OMap{}
	for index, row := range valueRange.Values {
		if index == 0 {
			continue
		}

		omap := new(OMap)
		if omap.FromRow(header, row) {
			maps = append(maps, omap)
		}
	}

	return maps, nil
}

func (h *AppHandler) readImageAspectRatio(ctx context.Context, imageURL string) (string, error) {
	response, err := http.Get(imageURL)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()
	config, _, err := image.DecodeConfig(response.Body)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%d:%d", config.Width, config.Height), nil
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

	channelSecret := consts.ChannelSecret(botName)
	channelAccessToken := consts.ChannelAccessToken(botName)

	bot, err := linebot.New(channelSecret, channelAccessToken)
	if err != nil {
		return err
	}

	lineEvents, err := bot.ParseRequest(c.Request())
	if err != nil {
		return err
	}

	for _, lineEvent := range lineEvents {
		switch lineEvent.Source.Type {
		case linebot.EventSourceTypeGroup:
			log.Printf("%s: %s\n", botName, lineEvent.Source.GroupID)
		}

		switch lineEvent.Type {
		case linebot.EventTypeMessage:
			switch lineMessage := lineEvent.Message.(type) {
			case *linebot.TextMessage:
				queries := strings.Split(lineMessage.Text, "　")
				switch queries[0] {
				case "設定":
					textMessage := linebot.NewTextMessage("名前の公開").
						WithQuickReplies(linebot.NewQuickReplyItems(
							linebot.NewQuickReplyButton("", linebot.NewMessageAction("する", "名前の公開：オン")),
							linebot.NewQuickReplyButton("", linebot.NewMessageAction("しない", "名前の公開：オフ")),
						))

					_, err = bot.ReplyMessage(lineEvent.ReplyToken, textMessage).Do()
					if err != nil {
						log.Printf(`Failed to reply message: message %s`, err.Error())
					}
				case "名前の公開：オン":
					_, err = firebase_.Client.Firestore.Doc("users/"+lineEvent.Source.UserID).Set(context.Background(), map[string]interface{}{
						"isHidden": false,
					}, firestore.MergeAll)

					if err != nil {
						log.Printf(`Failed to set isHidden = false: message %s`, err.Error())
					}

					textMessage := linebot.NewTextMessage("設定を更新しました！")
					_, err = bot.ReplyMessage(lineEvent.ReplyToken, textMessage).Do()
					if err != nil {
						log.Printf(`Failed to reply message: message %s`, err.Error())
					}
				case "名前の公開：オフ":
					_, err = firebase_.Client.Firestore.Doc("users/"+lineEvent.Source.UserID).Set(context.Background(), map[string]interface{}{
						"isHidden": true,
					}, firestore.MergeAll)

					if err != nil {
						log.Printf(`Failed to set isHidden = true: message %s`, err.Error())
					}

					textMessage := linebot.NewTextMessage("設定を更新しました！")
					_, err = bot.ReplyMessage(lineEvent.ReplyToken, textMessage).Do()
					if err != nil {
						log.Printf(`Failed to reply message: message %s`, err.Error())
					}
				// case "問題":
				// 	err = h.PushProblem(context.Background())
				// 	if err != nil {
				// 		log.Printf(`Failed to push problem: message %s`, err.Error())
				// 	}
				// case "解説":
				// 	err = h.PushEditorial(context.Background())
				// 	if err != nil {
				// 		log.Printf(`Failed to push editorial: message %s`, err.Error())
				// 	}
				case "地図":
					maps := []*OMap{}
					for _, omap := range h.maps {
						ok := (len(queries) == 1)
						if len(queries) > 1 {
							for _, q := range queries[1:] {
								ok = ok || (omap.Regulation == q)
							}
						}

						if ok {
							maps = append(maps, omap)
						}
					}

					omap := maps[rand.Intn(len(maps))]
					lines := append([]string{
						omap.Name,
						fmt.Sprintf("%d年度 %s (%s)", omap.Year, omap.Event, omap.Regulation),
					}, omap.URLs...)

					_, err = bot.ReplyMessage(lineEvent.ReplyToken, linebot.NewTextMessage(strings.Join(lines, "\n"))).Do()
					if err != nil {
						log.Printf(`Failed to reply message: message %s`, err.Error())
					}
				}
			}
		}
	}

	return c.NoContent(http.StatusOK)
}

func (h *AppHandler) LiffIndex(c echo.Context) error {
	return c.Render(http.StatusOK, "index.html", map[string]interface{}{})
}

func (h *AppHandler) LiffProblem(c echo.Context) error {
	problemID := c.Param("problemID")
	problem := h.problems[problemID]

	if problem == nil {
		return c.NoContent(http.StatusOK)
	}

	return c.Render(http.StatusOK, "problem.html", map[string]interface{}{
		"problemID": problemID,
		"text":      problem.Text,
		"imageURL":  problem.ProblemImageURL,
		"options":   problem.Options,
	})
}

func (h *AppHandler) LiffSubmit(c echo.Context) error {
	if h.todayProblem == nil {
		return c.NoContent(http.StatusOK)
	}

	type Parameter struct {
		ProblemID   string `json:"problemID"`
		UserID      string `json:"userID"`
		UserName    string `json:"userName"`
		UserGroupID string `json:"userGroupID"`
		Option      int    `json:"option"`
		Comment     string `json:"comment"`
	}

	param := new(Parameter)
	if err := c.Bind(param); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid parameter")
	}

	if h.todayProblem.ID != param.ProblemID {
		return c.NoContent(http.StatusOK)
	}

	answer := &Answer{
		ProblemID:    param.ProblemID,
		UserID:       param.UserID,
		UserName:     param.UserName,
		UserGroupID:  param.UserGroupID,
		UserIsHidden: false,
		Option:       param.Option,
		Comment:      param.Comment,
	}

	userSnapshot, err := firebase_.Client.Firestore.Doc("users/" + param.UserID).Get(context.Background())
	if err != nil {
		_, err := firebase_.Client.Firestore.Doc("users/"+param.UserID).Set(context.Background(), map[string]interface{}{
			"name":      param.UserName,
			"groupID":   param.UserGroupID,
			"createdAt": firestore.ServerTimestamp,
		}, firestore.MergeAll)

		if err != nil {
			log.Printf(`
				Failed to create user
					data: %s
					message %s
			`, json_.Marshal(param), err.Error())
		}
	} else {
		user := new(User)
		err = userSnapshot.DataTo(user)
		if err == nil {
			answer.UserIsHidden = user.IsHidden
		}
	}

	h.answers[param.UserID] = answer
	return c.NoContent(http.StatusOK)
}

func (h *AppHandler) UpdateMaps(ctx context.Context) error {
	maps, err := h.readOMaps(context.Background())
	if err != nil {
		return err
	}

	h.maps = maps
	return nil
}

func (h *AppHandler) PushProblem(ctx context.Context) error {
	problems, err := h.readProblems(ctx)
	if err != nil {
		return err
	}

	if len(problems) == 0 {
		return nil
	}

	problem := problems[rand.Intn(len(problems))]

	data := json_.ToMap(problem)
	data["createdAt"] = firestore.ServerTimestamp
	problemRef, _, err := firebase_.Client.Firestore.Collection("problems").Add(ctx, data)
	if err != nil {
		log.Printf(`
			Failed to create problem
				data: %s
				message %s
		`, json_.Marshal(problem), err.Error())

		return err
	}

	problem.ID = problemRef.ID
	h.todayProblem = problem
	h.problems[problem.ID] = problem
	h.answers = map[string]*Answer{}

	aspectRatio, err := h.readImageAspectRatio(ctx, problem.OriginalImageURL)
	if err != nil {
		aspectRatio = "1:1"
	}

	botNames := []string{
		"CHIMPANZEE", "CRAB", "RABBIT", "HAMSTER",
	}

	for _, botName := range botNames {
		channelAccessToken := consts.ChannelAccessToken(botName)
		groupID := consts.GroupID(botName)

		if channelAccessToken == "" || groupID == "" {
			continue
		}

		err = h.pushFlexMessage(
			ctx,
			channelAccessToken,
			groupID,
			"今日の1レッグ",
			consts.ProblemTemplatePath(),
			map[string]interface{}{
				"problemID":        problem.ID,
				"imageURL":         problem.OriginalImageURL,
				"imageAspectRatio": aspectRatio,
				"text":             problem.Text,
				"difficulty":       problem.Difficulty,
				"setter":           problem.Setter,
			},
		)

		if err != nil {
			log.Printf(`
				Failed to push problem
					botName: %s
					problemID: %d
					message: %s
			`, botName, problem.Index, err.Error())
		}
	}

	return h.setProblemSubmitted(ctx, problem.Index)
}

func (h *AppHandler) PushEditorial(ctx context.Context) error {
	if h.todayProblem == nil {
		return nil
	}

	type Result struct {
		Option        string   `json:"option"`
		Rate          int      `json:"rate"`
		Count         int      `json:"count"`
		IsMajority    bool     `json:"isMajority"`
		Answerers     []string `json:"answerers"`
		AnswerersText string   `json:"answerersText"`
	}

	type Comment struct {
		UserName string `json:"userName"`
		Text     string `json:"text"`
	}

	results := []*Result{}
	comments := []*Comment{}

	for _, option := range h.todayProblem.Options {
		results = append(results, &Result{
			Option: option,
			Rate:   0,
			Count:  0,
		})
	}

	maxCount := 0
	for _, answer := range h.answers {
		result := results[answer.Option]

		count := result.Count + 1
		result.Count = count
		if count > maxCount {
			maxCount = count
		}

		if !answer.UserIsHidden {
			result.Answerers = append(result.Answerers, answer.UserName)
		}

		results[answer.Option] = result

		if answer.Comment != "" {
			comments = append(comments, &Comment{
				UserName: answer.UserName,
				Text:     answer.Comment,
			})
		}

		data := json_.ToMap(answer)
		data["createdAt"] = firestore.ServerTimestamp
		answerRef, _, err := firebase_.Client.Firestore.Collection("answers").Add(context.Background(), data)
		if err != nil {
			log.Printf(`
				Failed to create answer
					data: %s
					message %s
			`, json_.Marshal(answer), err.Error())
			continue
		}

		answer.ID = answerRef.ID
	}

	for _, result := range results {
		if len(h.answers) > 0 {
			result.Rate = result.Count * 100 / len(h.answers)
		}
		result.IsMajority = (result.Count == maxCount)

		// Shuffle
		answerers := func(a []string) []string {
			for i := range a {
				j := rand.Intn(i + 1)
				a[i], a[j] = a[j], a[i]
			}
			return a
		}(result.Answerers)

		elseCount := result.Count
		if len(answerers) > 10 {
			elseCount -= 10
		} else {
			elseCount -= len(answerers)
		}

		if len(answerers) == 0 {
			if result.Count > 0 {
				result.AnswerersText = fmt.Sprintf("回答者%d人", elseCount)
			} else {
				result.AnswerersText = "回答者なし"
			}
		} else {
			text := strings.Join(answerers[:result.Count-elseCount], "、")
			if elseCount > 0 {
				text = text + fmt.Sprintf(" 他%d人", elseCount)
			}
			result.AnswerersText = text
		}
	}

	aspectRatio, err := h.readImageAspectRatio(ctx, h.todayProblem.EditorialImageURL)
	if err != nil {
		aspectRatio = "1:1"
	}

	args := json_.ToMap(map[string]interface{}{
		"imageURL":         h.todayProblem.EditorialImageURL,
		"imageAspectRatio": aspectRatio,
		"text":             h.todayProblem.Editorial,
		"count":            len(h.answers),
		"results":          results,
		"comments":         comments,
	})

	botNames := []string{
		"CHIMPANZEE", "CRAB", "RABBIT", "HAMSTER",
	}

	for _, botName := range botNames {
		channelAccessToken := consts.ChannelAccessToken(botName)
		groupID := consts.GroupID(botName)

		if channelAccessToken == "" || groupID == "" {
			continue
		}

		err = h.pushFlexMessage(
			ctx,
			channelAccessToken,
			groupID,
			"今日の1レッグ（解説）",
			consts.EditorialTemplatePath(),
			args,
		)

		if err != nil {
			log.Printf(`
				Failed to push editorial
					botName: %s
					problemIndex: %d
					message: %s
			`, botName, h.todayProblem.Index, err.Error())
		}
	}

	return nil
}
