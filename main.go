package main

import (
	"context"
	"html/template"
	"io"
	"log"
	"time"

	"github.com/kawasin73/htask/cron"
	"github.com/kuolc/oneLeg/consts"
	"github.com/kuolc/oneLeg/scheduler"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	h := &AppHandler{
		problems: make(map[ProblemID]*Problem),
		answers:  make(map[UserID]*Answer),
	}

	scheduler.Set("update_maps", func(cr *cron.Cron) *scheduler.Job {
		cancel, _ := cr.Every(1).Day().At(consts.UpdateMapsAt()).Run(func() {
			err := h.UpdateMaps(context.Background())
			if err != nil {
				log.Printf(`
					Failed to update maps
						message %s
				`, err.Error())
			}
		})

		return &scheduler.Job{Cancel: cancel}
	})

	scheduler.Set("push_problem", func(cr *cron.Cron) *scheduler.Job {
		cancel, _ := cr.Every(1).Day().At(consts.PushProblemAt()).Run(func() {
			weekday := time.Now().Weekday()
			if !(weekday >= 1 && weekday <= 5) {
				return
			}

			err := h.PushProblem(context.Background())
			if err != nil {
				log.Printf(`
					Failed to push problem
						message %s
				`, err.Error())
			}
		})

		return &scheduler.Job{Cancel: cancel}
	})

	scheduler.Set("push_editorial", func(cr *cron.Cron) *scheduler.Job {
		cancel, _ := cr.Every(1).Day().At(consts.PushEditorialAt()).Run(func() {
			err := h.PushEditorial(context.Background())
			if err != nil {
				log.Printf(`
					Failed to push editorial
						message %s
				`, err.Error())
			}
		})

		return &scheduler.Job{Cancel: cancel}
	})

	err := h.UpdateMaps(context.Background())
	if err != nil {
		log.Printf(`
			Failed to update maps
				message %s
		`, err.Error())
	}

	e := echo.New()

	e.Renderer = &Template{
		templates: template.Must(template.ParseGlob("public/views/*.html")),
	}

	e.Use(middleware.Recover())
	e.POST("/webhook/:botName", h.Webhook)
	e.GET("/liff", h.LiffIndex)
	e.GET("/liff/problems/:problemID", h.LiffProblem)
	e.POST("/liff", h.LiffSubmit)

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		e.DefaultHTTPErrorHandler(err, c)
		print(err.Error())
	}

	if !consts.IsLocal() {
		e.Pre(middleware.HTTPSWWWRedirect())
		e.StartTLS(":443", consts.TLSCertificatePath(), consts.TLSPrivateKeyPath())
	} else {
		e.Start(":80")
	}
}
