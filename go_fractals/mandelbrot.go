package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"math/cmplx"
	"os"
	"runtime"
	"sync"
	"time"
)

var IMAGE_WIDTH, IMAGE_HEIGHT = 8 * 1024, 8 * 1024
var X_MIN, X_MAX, Y_MIN, Y_MAX = -2.0, 1.0, -1.5, 1.5
var MAX_ITER = 1000
var MAX_MAGNITUDE = 10

var xScale = (X_MAX - X_MIN) / float64(IMAGE_WIDTH)
var yScale = (Y_MAX - Y_MIN) / float64(IMAGE_HEIGHT)

var colorCache = make([]color.RGBA, MAX_ITER+1)

func inMandelbrotSet(x float64, y float64) (bool, float64) {
	z := complex(0, 0)
	c := complex(x, y)

	for i := 0; i < MAX_ITER; i++ {
		z = z*z + c
		if cmplx.Abs(z) > float64(MAX_MAGNITUDE) {
			return false, float64(i) / float64(MAX_ITER)
		}
	}

	return true, 0.0
}

func colorFromValue(value float64) color.RGBA {
	// Apply logarithmic scaling to the value
	scaledValue := math.Log1p(value*99) / math.Log1p(99)

	// Viridis-like color map with logarithmic scaling
	r := uint8(255 * (1 - math.Sin(scaledValue*math.Pi)))
	g := uint8(255 * math.Sin(scaledValue*math.Pi))
	b := uint8(255 * (1 - math.Cos(scaledValue*math.Pi)))
	return color.RGBA{r, g, b, 255}
}

func main() {
	start := time.Now()

	// Pre calculate colors
	for i := range MAX_ITER {
		value := float64(i) / float64(MAX_ITER)
		colorCache[i] = colorFromValue(value)
	}

	type Pixel struct {
		px, py int
		col    color.RGBA
	}

	// Buffered channel to reduce goroutine blocking
	pixelChan := make(chan Pixel, IMAGE_WIDTH*IMAGE_HEIGHT/runtime.NumCPU())

	numWorkers := runtime.NumCPU()
	var wg sync.WaitGroup

	// Create image and color pixels
	img := image.NewRGBA(image.Rect(0, 0, IMAGE_WIDTH, IMAGE_HEIGHT))
	rowsPerWorker := IMAGE_HEIGHT / numWorkers

	for workerID := range numWorkers {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Calculate this worker's row range
			startRow := id * rowsPerWorker
			endRow := startRow + rowsPerWorker

			if id == numWorkers {
				endRow = IMAGE_HEIGHT
			}

			for px := startRow; px < endRow; px++ {
				for py := 0; py < IMAGE_HEIGHT; py++ {
					x := X_MIN + float64(px)*xScale
					y := Y_MIN + float64(py)*yScale

					inSet, val := inMandelbrotSet(x, y)

					var col color.RGBA
					if inSet {
						col = color.RGBA{0, 0, 0, 255} // Black for points in the set
					} else {
						col = colorCache[uint8(val*float64(MAX_ITER))]
					}
					pixelChan <- Pixel{px, py, col}
				}
			}
		}(workerID)

	}

	go func() {
		wg.Wait()
		close(pixelChan)
	}()

	for pixel := range pixelChan {
		img.Set(pixel.px, pixel.py, pixel.col)
	}

	// Save output image
	file, err := os.Create("output.png")
	if err != nil {
		log.Fatalf("Failed to create output file %v", err)
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		log.Fatalf("Failed to encode image: %v", err)
	}

	elapsed := time.Since(start)
	fmt.Println("Image created!")
	fmt.Printf("Mandelbrot generation took %v\n", elapsed)
}
