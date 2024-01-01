// main.go
package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"speech-to-text/helper"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type TemplateRenderer struct {
	templates *template.Template
}

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	e := echo.New()

	if err := godotenv.Load(".env"); err != nil {
		log.Fatalf("error while reading .env file")
	}

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Register HTML template renderer
	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}
	e.Renderer = renderer

	// Routes
	e.GET("/", homeHandler)
	e.POST("/upload", uploadHandler)

	// Start server
	e.Start(":8080")
}

func homeHandler(c echo.Context) error {
	return c.Render(http.StatusOK, "index.html", nil)
}

func uploadHandler(c echo.Context) error {
	// Get the uploaded MP3 file
	file, err := c.FormFile("mp3")
	if err != nil {
		return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"message": "Failed to get the file"})
	}

	// Open the file
	src, err := file.Open()
	if err != nil {
		return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"message": "Failed to open the file"})
	}
	defer src.Close()

	fileFormat := file.Filename[len(file.Filename)-3:]
	formats := []string{"mp3", "flac", "m4a", "mp4", "mpeg", "mpga", "oga", "ogg", "wav", "webm"}
	if !helper.Contains(formats, fileFormat) {
		_, err := os.Create("data/" + file.Filename)
		if err != nil {
			log.Println(err)
		}
		return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"message": fmt.Sprintf("file format should be only: %v", formats)})
	}

	// Save the file to the "uploads" directory
	dst, err := os.Create("uploads/" + uuid.NewString() + "." + fileFormat)
	if err != nil {
		return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"message": "Failed to save the file"})
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"message": "Failed to save the file"})
	}

	// Call your speech-to-text logic
	transcribedText, err := helper.SpeechToText(dst.Name())
	if err != nil {
		return c.Render(http.StatusInternalServerError, "error.html", map[string]string{"message": "Failed to transcribe speech"})
	}

	// Delete the uploaded file
	if err := os.Remove(dst.Name()); err != nil {
		log.Println("Error deleting file:", err)
	}

	return c.Render(http.StatusOK, "result.html", map[string]string{"text": transcribedText})
}
