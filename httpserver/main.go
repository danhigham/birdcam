package main

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"net/http"

	"cloud.google.com/go/storage"
	"github.com/aymerick/raymond"
	"github.com/gobuffalo/uuid"
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
	ctx := context.Background()

	secret := c.Param("secret")
	if secret != os.Getenv("SNAPSHOT_SECRET") {
		return c.String(http.StatusUnauthorized, "Unauthorized")
	}

	//projectID := os.Getenv("GCP_PROJECT_ID")
	bucketName := os.Getenv("BUCKET_NAME")
	snapshotURL := os.Getenv("SNAPSHOT_URL")
	snapshotUser := os.Getenv("SNAPSHOT_USER")
	snapshotPassword := os.Getenv("SNAPSHOT_PASSWORD")

	client, err := storage.NewClient(ctx)
	if err != nil {
		return c.String(http.StatusInternalServerError, fmt.Sprintf("%v", err))
	}

	bucket := client.Bucket(bucketName)

	for i := 0; i < 5; i++ {
		resp, err := digestGet(snapshotURL, snapshotUser, snapshotPassword)

		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("%v", err))
		}

		defer resp.Body.Close()

		imgBytes, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("%v", err))
		}

		id, _ := uuid.NewV4()

		filename := fmt.Sprintf("snapshot-%s.jpg", id)
		wc := bucket.Object(filename).NewWriter(ctx)
		wc.ContentType = resp.Header.Get("Content-Type")

		if _, err := wc.Write(imgBytes); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("%v", err))
		}

		if err := wc.Close(); err != nil {
			return c.String(http.StatusInternalServerError, fmt.Sprintf("%v", err))
		}
		time.Sleep(500 * time.Millisecond)
	}
	return c.HTML(http.StatusOK, "OK")
}

func digestGet(url string, username string, password string) (*http.Response, error) {
	method := "GET"
	req, err := http.NewRequest(method, url, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	digestParts := digestParts(resp)
	digestParts["uri"] = "/cgi-bin/snapshot.cgi"
	digestParts["method"] = method
	digestParts["username"] = username
	digestParts["password"] = password
	req, err = http.NewRequest(method, url, nil)
	req.Header.Set("Authorization", getDigestAuthrization(digestParts))

	return client.Do(req)
}

func digestParts(resp *http.Response) map[string]string {
	result := map[string]string{}
	if len(resp.Header["Www-Authenticate"]) > 0 {
		wantedHeaders := []string{"nonce", "realm", "qop"}
		responseHeaders := strings.Split(resp.Header["Www-Authenticate"][0], ",")
		for _, r := range responseHeaders {
			for _, w := range wantedHeaders {
				if strings.Contains(r, w) {
					result[w] = strings.Split(r, `"`)[1]
				}
			}
		}
	}
	return result
}

func getMD5(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func getCnonce() string {
	b := make([]byte, 8)
	io.ReadFull(rand.Reader, b)
	return fmt.Sprintf("%x", b)[:16]
}

func getDigestAuthrization(digestParts map[string]string) string {
	d := digestParts
	ha1 := getMD5(d["username"] + ":" + d["realm"] + ":" + d["password"])
	ha2 := getMD5(d["method"] + ":" + d["uri"])
	nonceCount := 00000001
	cnonce := getCnonce()
	response := getMD5(fmt.Sprintf("%s:%s:%v:%s:%s:%s", ha1, d["nonce"], nonceCount, cnonce, d["qop"], ha2))
	authorization := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", cnonce="%s", nc="%v", qop="%s", response="%s"`,
		d["username"], d["realm"], d["nonce"], d["uri"], cnonce, nonceCount, d["qop"], response)
	return authorization
}
