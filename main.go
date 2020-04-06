package main

import (
	"github.com/aruga-dev/arugaONE-API/util/consts"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func errorHandler(err error, c echo.Context) {

}

func main() {
	h := &AppHandler{}

	e := echo.New()
	e.Use(middleware.Recover())
	e.POST("/push", h.PushMessage)
	e.POST("/webhook/:botName", h.Webhook)

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
