// What it does:
//
// This example detects motion using a delta threshold from the first frame,
// and then finds contours to determine where the object is located.
//
// Very loosely based on Adrian Rosebrock code located at:
// http://www.pyimagesearch.com/2015/06/01/home-surveillance-and-motion-detection-with-the-raspberry-pi-python-and-opencv/
//
// How to run:
//
// 		go run ./cmd/motion-detect/main.go 0
//

package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"net/http"
	"os"

	"github.com/hybridgroup/mjpeg"
	"gocv.io/x/gocv"
)

const MinimumArea = 1500

type BirdStream struct {
	Stream  *mjpeg.Stream
	Channel chan gocv.Mat
}

func main() {

	rtspstream := os.Getenv("INPUT_STREAM")
	callbackURL := os.Getenv("CALLBACK_URL")

	webcam, err := gocv.VideoCaptureFile(rtspstream)
	if err != nil {
		fmt.Printf("Error opening stream: %v\n", rtspstream)
		return
	}
	defer webcam.Close()

	img := gocv.NewMat()
	defer img.Close()

	mask, err := loadImage("./mask.jpg")
	if err != nil {
		fmt.Printf("Error opening mask")
		return
	}

	testMidX := 320
	testMidY := 240

	maskR := mask[0].GetUCharAt(testMidY, testMidX)
	maskG := mask[1].GetUCharAt(testMidY, testMidX)
	maskB := mask[2].GetUCharAt(testMidY, testMidX)

	fmt.Printf("Midpoint mask test R: %v, G: %v, B: %v\n", maskR, maskG, maskB)

	imgDelta := gocv.NewMat()
	defer imgDelta.Close()

	imgThresh := gocv.NewMat()
	defer imgThresh.Close()

	mog2 := gocv.NewBackgroundSubtractorMOG2()
	defer mog2.Close()

	plainStream := BirdStream{Stream: mjpeg.NewStream(), Channel: make(chan gocv.Mat)}
	trackingStream := BirdStream{Stream: mjpeg.NewStream(), Channel: make(chan gocv.Mat)}

	go func() {
		http.Handle("/stream/no-tracking", plainStream.Stream)
		http.Handle("/stream/tracking", trackingStream.Stream)
		log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
	}()

	go capture(plainStream)
	go capture(trackingStream)

	status := "Ready"

	fmt.Printf("Start reading camera device: %v\n", rtspstream)
	for {
		if ok := webcam.Read(&img); !ok {
			fmt.Printf("Error cannot read device %d\n", rtspstream)
			return
		}
		if img.Empty() {
			continue
		}

		plainStream.Channel <- img

		status = "Ready"
		statusColor := color.RGBA{0, 255, 0, 0}

		// first phase of cleaning up image, obtain foreground only
		mog2.Apply(img, &imgDelta)

		// remaining cleanup of the image to use for finding contours.
		// first use threshold
		gocv.Threshold(imgDelta, &imgThresh, 25, 255, gocv.ThresholdBinary)

		// then dilate
		kernel := gocv.GetStructuringElement(gocv.MorphRect, image.Pt(3, 3))
		defer kernel.Close()
		gocv.Dilate(imgThresh, &imgThresh, kernel)

		// now find contours
		contours := gocv.FindContours(imgThresh, gocv.RetrievalExternal, gocv.ChainApproxSimple)
		for _, c := range contours {
			area := gocv.ContourArea(c)
			if area < MinimumArea {
				continue
			}

			rect := gocv.BoundingRect(c)
			midX := rect.Min.X + (rect.Dx() / 2)
			midY := rect.Min.Y + (rect.Dy() / 2)

			maskR := mask[0].GetUCharAt(midY, midX)
			maskG := mask[1].GetUCharAt(midY, midX)
			maskB := mask[2].GetUCharAt(midY, midX)

			if (maskR + maskG + maskB) > 0 {
				break
			}

			status = "Motion detected"
			statusColor = color.RGBA{255, 0, 0, 0}

			gocv.Rectangle(&img, rect, color.RGBA{255, 0, 0, 0}, 2)

			if callbackURL != "" {
				http.Get(callbackURL)
			}
		}

		gocv.PutText(&img, status, image.Pt(10, 20), gocv.FontHersheyPlain, 1.2, statusColor, 2)
		trackingStream.Channel <- img
	}

}

func capture(birdStream BirdStream) {
	for {
		m := <-birdStream.Channel
		buf, _ := gocv.IMEncode(".jpg", m)
		birdStream.Stream.UpdateJPEG(buf)
	}
}

func loadImage(filename string) ([]gocv.Mat, error) {
	img := gocv.IMRead(filename, gocv.IMReadColor)
	if img.Empty() {
		return []gocv.Mat{}, errors.New(fmt.Sprintf("Error reading image from: %v", filename))
	}

	mat := gocv.Split(img)
	return mat, nil
}

func writeMatToFile(mat gocv.Mat, filename string) {
	target, err := mat.ToImage()
	if err != nil {
		panic(err)
	}
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	jpeg.Encode(f, target, nil)
}
