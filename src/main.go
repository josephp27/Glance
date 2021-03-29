package main

import (
	"bytes"
	"fmt"
	"github.com/nfnt/resize"
	"image"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gen2brain/x264-go"
	"github.com/kbinani/screenshot"
)

const h264SupportedProfile = "3.1"

//findBestSizeForH264Profile finds the best match given the size constraint and H264 profile
func findBestSizeForH264Profile(profile string, constraints image.Point) (image.Point, error) {
	profileSizes := map[string][]image.Point{
		"3.1": []image.Point{
			image.Point{1280, 720},
			image.Point{720, 576},
			image.Point{720, 480},
		},
	}
	if sizes, exists := profileSizes[profile]; exists {
		minRatioDiff := math.MaxFloat64
		var minRatioSize image.Point
		for _, size := range sizes {
			if size == constraints {
				return size, nil
			}
			lowerRes := size.X < constraints.X && size.Y < constraints.Y
			hRatio := float64(constraints.X) / float64(size.X)
			vRatio := float64(constraints.Y) / float64(size.Y)
			ratioDiff := math.Abs(hRatio - vRatio)
			if lowerRes && (ratioDiff) < 0.0001 {
				return size, nil
			} else if ratioDiff < minRatioDiff {
				minRatioDiff = ratioDiff
				minRatioSize = size
			}
		}
		return minRatioSize, nil
	}
	return image.Point{}, fmt.Errorf("Profile %s not supported", profile)
}

func main() {
	buf := bytes.NewBuffer(make([]byte, 0))

	bounds := screenshot.GetDisplayBounds(0)

	sourceSize := image.Point{
		bounds.Dx(),
		bounds.Dy(),
	}

	realSize, err := findBestSizeForH264Profile(h264SupportedProfile, sourceSize)

	opts := &x264.Options{
		Width:     realSize.X,
		Height:    realSize.Y,
		FrameRate: 60,
		Tune:      "zerolatency",
		Preset:    "ultrafast",
		Profile:   "baseline",
		//LogLevel:  x264.LogDebug,
	}

	enc, err := x264.NewEncoder(buf, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	defer enc.Close()

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM)

	now := time.Now()
	secs := now.Unix()
	count := 0
	for {

		count += 1
		img, err := screenshot.CaptureRect(bounds)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			continue
		}

		rgba := resize.Resize(uint(bounds.Dx()/10), uint(bounds.Dy()/10), img, resize.Lanczos3).(*image.RGBA)

		err = enc.Encode(rgba)
		err = enc.Flush()
		//if err != nil {
		//	fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		//}

		elapsed := time.Now().Unix() - secs
		if elapsed > 0 {
			println(int64(count)/elapsed, count, elapsed)
		}

	}
}
