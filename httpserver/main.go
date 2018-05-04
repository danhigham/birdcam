package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"net/http"

	"github.com/aymerick/raymond"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

func main() {

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Static("./static"))

	e.GET("/", index)
	e.GET("/snapshot/:secret", snapshot)

	// Start server
	e.Logger.Fatal(e.Start(":8080"))

}

func index(c echo.Context) error {
	tmplCtx := map[string]string{
		"stream_url": os.Getenv("STREAM_URL"),
	}

	b, err := ioutil.ReadFile("./templates/index.html") // just pass the file name
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%s", err))
	}

	result, err := raymond.Render(string(b), tmplCtx)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%s", err))
	}

	return c.HTML(http.StatusOK, result)
}

func snapshot(c echo.Context) error {
	secret := c.Param("secret")

	if secret != os.Getenv("SNAPSHOT_SECRET") {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	return c.HTML(http.StatusOK, "OK")
}
