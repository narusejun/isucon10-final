// +build !release

package embed

import (
	"net/http"
	"net/http/pprof"

	"github.com/isucon/isucon10-final/webapp/golang/embed/transport"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

func EmbedDebugServer(addr string) {
	e := echo.New()
	EnableLogging(e)
	EnablePProf(e)
	EnableLogTransport(e)
	go func() {
		e.Start(addr)
	}()
}

func EnableLogTransport(e *echo.Echo) {
	g := e.Group("/debug/log")
	g.Any("/app", transport.NewTailHandler("./app.log").Handle)
	g.Any("/envoy", transport.NewTailHandler("/var/log/envoy/access.log").Handle)
}

func EnablePProf(e *echo.Echo) {
	g := e.Group("/debug/pprof")
	g.Any("/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	g.Any("/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	g.Any("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	g.Any("/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))
	g.Any("/*", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
}

func EnableLogging(e *echo.Echo) {
	e.Debug = true
	e.Logger.SetLevel(log.DEBUG)
	e.Use(middleware.Logger())
}
