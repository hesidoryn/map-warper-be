package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/labstack/echo"

	b64 "encoding/base64"
)

const (
	tifTranslated = "/tmp/translated.tif"
	tifWarped     = "/tmp/warped.tif"
)

type data struct {
	Points      []point `json:"points"`
	Filename    string  `json:"filename"`
	ImageBase64 string  `json:"imageBase64"`
}

type point struct {
	Image imageCoords `json:"image"`
	Geo   geoCoords   `json:"geo"`
}

type imageCoords struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type geoCoords struct {
	Lat  float64 `json:"lat"`
	Long float64 `json:"long"`
}

func main() {
	addr := ":" + os.Getenv("PORT")

	e := echo.New()
	e.POST("/", uploadHandler)
	e.Start(addr)
}

func uploadHandler(c echo.Context) error {
	d := data{}
	err := c.Bind(&d)
	if err != nil {
		log.Println(err)
		c.Error(err)
	}

	bytes, err := b64.StdEncoding.DecodeString(d.ImageBase64)
	if err != nil {
		log.Println(err)
		c.Error(err)
	}

	frmt := strings.Split(d.Filename, ".")[1]
	newFilename := fmt.Sprintf("/tmp/new.%s", frmt)
	f, err := os.Create(newFilename)
	if err != nil {
		log.Println(err)
		c.Error(err)
	}
	defer f.Close()

	f.Write(bytes)

	err = gdalTranslate(newFilename, d.Points)
	if err != nil {
		log.Println("gdal_translate")
		log.Println(err)
		c.Error(err)
	}

	err = gdalWarp()
	if err != nil {
		log.Println("gdalwarp")
		log.Println(err)
		c.Error(err)
	}

	err = uploadToS3()
	if err != nil {
		log.Println("uploadToS3")
		log.Println(err)
		c.Error(err)
	}

	return c.File(tifWarped)
}

func gdalTranslate(file string, points []point) error {
	gdalTr := "./gdal_translate"
	args := []string{"-of", "GTiff"}
	for _, p := range points {
		x, y := strconv.Itoa(p.Image.X), strconv.Itoa(p.Image.Y)
		lat, long := strconv.FormatFloat(p.Geo.Lat, 'f', 6, 64), strconv.FormatFloat(p.Geo.Long, 'f', 6, 64)
		args = append(args, "-gcp", y, x, long, lat)
	}
	args = append(args, file, tifTranslated)

	cmd := exec.Command(gdalTr, args...)
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	fmt.Println(stdout.String())
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return err
	}
	return nil
}

func gdalWarp() error {
	os.Setenv("GDAL_DATA", "/var/task/data")

	warp := "./gdalwarp"
	args := []string{"-t_srs", "EPSG:4326"}
	args = append(args, tifTranslated, tifWarped)
	log.Println(args)

	cmd := exec.Command(warp, args...)
	var stderr bytes.Buffer
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	fmt.Println(stdout.String())
	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
		return err
	}
	return nil
}

func uploadToS3() error {
	log.Println("uploadToS3: START")
	// The session the S3 Uploader will use
	sess := session.Must(session.NewSession())
	log.Println("uploadToS3: after session must")

	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)
	log.Println("uploadToS3: after uploader creating")

	f, err := os.Open(tifWarped)
	if err != nil {
		log.Println("uploadToS3: file open error")
		log.Println(err)
		return err
	}

	// Upload the file to S3.
	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("map-warper-be-test"),
		Key:    aws.String(f.Name()),
		Body:   f,
	})
	log.Println("uploadToS3: after upload")
	log.Println(err)
	log.Println(result)
	return err
}
