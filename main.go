package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"perlin_noise/perlin"
	"perlin_noise/poi"
)

const (
	width, height = 512, 512
)

var (
	deepWaterColor = color.RGBA{R: 25, G: 70, B: 120, A: 255}
	waterColor     = color.RGBA{R: 50, G: 150, B: 200, A: 255}
	shoreColor     = color.RGBA{R: 240, G: 230, B: 140, A: 255}
	landColor      = color.RGBA{R: 80, G: 180, B: 80, A: 255}
	mountainColor  = color.RGBA{R: 120, G: 100, B: 80, A: 255}
	highMountainColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	highLandColor     = color.RGBA{R: 100, G: 150, B: 100, A: 255}
	poiColor       = color.RGBA{R: 255, G: 0, B: 0, A: 255}
)

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func main() {
	// seed the global rand for the randomize button
	rand.Seed(time.Now().UnixNano())

	myApp := app.New()
	myApp.Settings().SetTheme(theme.DarkTheme())
	myWindow := myApp.NewWindow("Perlin Noise Generator")
	myWindow.Resize(fyne.NewSize(1024, 768))

	// Default parameters (tweak to taste)
	var seed int64 = 12345
	var scale float64 = 0.006
	var octavesFloat float64 = 5.0
	var persistence float64 = 0.5
	var lacunarity float64 = 2.0

	var continentFreq float64 = 0.004
	var continentOctavesFloat float64 = 3.0
	var continentWeight float64 = 0.6

	var falloff float64 = 1.8
	var falloffWeight float64 = 0.6

	var seaLevel float64 = 0.45
	var minDistance int64 = 25

	var flowScale float64 = 0.002
	var flowStrength float64 = 15.0

	var mutex sync.Mutex
	var isGenerating bool

	// Shared image (always replaced atomically)
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	imageCanvas := canvas.NewImageFromImage(img)
	imageCanvas.SetMinSize(fyne.NewSize(width, height))
	imageCanvas.FillMode = canvas.ImageFillOriginal

	// Labels
	seedLabel := widget.NewLabel(fmt.Sprintf("Seed: %d", seed))
	scaleLabel := widget.NewLabel(fmt.Sprintf("Scale: %.4f", scale))
	octavesLabel := widget.NewLabel(fmt.Sprintf("Octaves: %.0f", octavesFloat))
	persistenceLabel := widget.NewLabel(fmt.Sprintf("Persistence: %.2f", persistence))
	lacunarityLabel := widget.NewLabel(fmt.Sprintf("Lacunarity: %.2f", lacunarity))

	continentFreqLabel := widget.NewLabel(fmt.Sprintf("Continent Freq: %.4f", continentFreq))
	continentOctavesLabel := widget.NewLabel(fmt.Sprintf("Continent Octaves: %.0f", continentOctavesFloat))
	continentWeightLabel := widget.NewLabel(fmt.Sprintf("Continent Weight: %.2f", continentWeight))

	falloffLabel := widget.NewLabel(fmt.Sprintf("Falloff: %.2f", falloff))
	falloffWeightLabel := widget.NewLabel(fmt.Sprintf("Falloff Weight: %.2f", falloffWeight))

	seaLevelLabel := widget.NewLabel(fmt.Sprintf("Sea Level: %.2f", seaLevel))
	minDistanceLabel := widget.NewLabel(fmt.Sprintf("Min. Distance: %d", minDistance))

	flowScaleLabel := widget.NewLabel(fmt.Sprintf("Flow Scale: %.4f", flowScale))
	flowStrengthLabel := widget.NewLabel(fmt.Sprintf("Flow Strength: %.2f", flowStrength))

	// updateImage (background-generation safe)
	updateImage := func() {
		// create a fresh out image locally to avoid mutating the shared img while UI reads it
		out := image.NewRGBA(image.Rect(0, 0, width, height))

		// local perlin instance
		p := perlin.NewPerlin(seed)

		centerX := float64(width) / 2.0
		centerY := float64(height) / 2.0
		maxDist := math.Hypot(centerX, centerY)

		octaves := int(octavesFloat)
		continentOctaves := int(continentOctavesFloat)

		// prepare map for POIs
		noiseMap := make(map[poi.Point]float64, width*height)

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				// signed flow in [-1,1]
				flowXRaw, flowYRaw := p.NoiseFlow(float64(x), float64(y), flowScale)
				dx := flowXRaw * flowStrength
				dy := flowYRaw * flowStrength

				px := float64(x) + dx
				py := float64(y) + dy

				// local detail
				localRaw := p.FBM2DRaw(px, py, scale, octaves, persistence, lacunarity)
				// large-scale continent mask
				continentRaw := p.FBM2DRaw(float64(x), float64(y), continentFreq, continentOctaves, 0.5, 2.0)

				combinedRaw := localRaw*(1.0-continentWeight) + continentRaw*continentWeight
				combined := (combinedRaw + 1.0) * 0.5

				dist := math.Hypot(float64(x)-centerX, float64(y)-centerY)
				falloffVal := math.Pow(dist/maxDist, falloff) * falloffWeight

				noiseValue := clamp01(combined - falloffVal)

				// store for POIs
				noiseMap[poi.Point{X: x, Y: y}] = noiseValue

				// color
				var pixelColor color.Color
				if noiseValue < seaLevel-0.15 {
					pixelColor = deepWaterColor
				} else if noiseValue < seaLevel {
					pixelColor = waterColor
				} else if noiseValue < seaLevel+0.04 {
					pixelColor = shoreColor
				} else if noiseValue < seaLevel+0.10 {
					pixelColor = landColor
				} else if noiseValue < seaLevel+0.20 {
					pixelColor = highLandColor
				} else if noiseValue < seaLevel+0.30 {
					pixelColor = mountainColor
				} else {
					pixelColor = highMountainColor
				}
				out.Set(x, y, pixelColor)
			}
		}

		// POIs drawn onto out
		// Each POI run needs its own source to be threadsafe
		poiRand := rand.New(rand.NewSource(seed))
		pois, _ := poi.PoissonDisk(minDistance, width, height, poiRand, noiseMap, seaLevel)
		for _, pnt := range pois {
			for i := -1; i <= 1; i++ {
				for j := -1; j <= 1; j++ {
					xx := pnt.X + i
					yy := pnt.Y + j
					if xx >= 0 && xx < width && yy >= 0 && yy < height {
						out.Set(xx, yy, poiColor)
					}
				}
			}
		}

		// swap into shared img under mutex
		mutex.Lock()
		img = out
		mutex.Unlock()

		// Schedule UI update on the main GUI thread using fyne.Do
		fyne.Do(func() {
			imageCanvas.Image = img
			imageCanvas.Refresh()
		})
	}

	// safe trigger
	triggerUpdate := func() {
		mutex.Lock()
		defer mutex.Unlock()
		if !isGenerating {
			isGenerating = true
			go func() {
				updateImage()
				mutex.Lock()
				isGenerating = false
				mutex.Unlock()
			}()
		}
	}

	// Seed slider (no automatic generation on change)
	seedSlider := widget.NewSlider(0, 100000)
	seedSlider.Step = 1
	seedSlider.Value = float64(seed)
	seedSlider.OnChanged = func(v float64) {
		seed = int64(v)
		seedLabel.SetText(fmt.Sprintf("Seed: %d", seed))
		// note: no triggerUpdate() here to avoid generating on every drag
	}

	// Randomize Seed button - sets a new seed and triggers a single generation
	randomSeedBtn := widget.NewButton("Randomize Seed & Generate", func() {
		seed = rand.Int63n(100000)
		seedLabel.SetText(fmt.Sprintf("Seed: %d", seed))
		// also update slider position to reflect new seed
		seedSlider.SetValue(float64(seed))
		triggerUpdate()
	})

	// Scale slider
	scaleSlider := widget.NewSlider(0.001, 0.02)
	scaleSlider.Step = 0.0005
	scaleSlider.Value = scale
	scaleSlider.OnChanged = func(v float64) {
		scale = v
		scaleLabel.SetText(fmt.Sprintf("Scale: %.4f", scale))
		triggerUpdate()
	}

	// Octaves slider
	octavesSlider := widget.NewSlider(1, 8)
	octavesSlider.Step = 1
	octavesSlider.Value = octavesFloat
	octavesSlider.OnChanged = func(v float64) {
		octavesFloat = v
		octavesLabel.SetText(fmt.Sprintf("Octaves: %.0f", octavesFloat))
		triggerUpdate()
	}

	// Persistence slider
	persistenceSlider := widget.NewSlider(0.1, 0.9)
	persistenceSlider.Step = 0.01
	persistenceSlider.Value = persistence
	persistenceSlider.OnChanged = func(v float64) {
		persistence = v
		persistenceLabel.SetText(fmt.Sprintf("Persistence: %.2f", persistence))
		triggerUpdate()
	}

	// Lacunarity slider
	lacunaritySlider := widget.NewSlider(1.5, 3.0)
	lacunaritySlider.Step = 0.05
	lacunaritySlider.Value = lacunarity
	lacunaritySlider.OnChanged = func(v float64) {
		lacunarity = v
		lacunarityLabel.SetText(fmt.Sprintf("Lacunarity: %.2f", lacunarity))
		triggerUpdate()
	}

	// Continent sliders
	continentFreqSlider := widget.NewSlider(0.0005, 0.02)
	continentFreqSlider.Step = 0.0005
	continentFreqSlider.Value = continentFreq
	continentFreqSlider.OnChanged = func(v float64) {
		continentFreq = v
		continentFreqLabel.SetText(fmt.Sprintf("Continent Freq: %.4f", continentFreq))
		triggerUpdate()
	}

	continentOctavesSlider := widget.NewSlider(1, 6)
	continentOctavesSlider.Step = 1
	continentOctavesSlider.Value = continentOctavesFloat
	continentOctavesSlider.OnChanged = func(v float64) {
		continentOctavesFloat = v
		continentOctavesLabel.SetText(fmt.Sprintf("Continent Octaves: %.0f", continentOctavesFloat))
		triggerUpdate()
	}

	continentWeightSlider := widget.NewSlider(0.0, 1.0)
	continentWeightSlider.Step = 0.01
	continentWeightSlider.Value = continentWeight
	continentWeightSlider.OnChanged = func(v float64) {
		continentWeight = v
		continentWeightLabel.SetText(fmt.Sprintf("Continent Weight: %.2f", continentWeight))
		triggerUpdate()
	}

	// Falloff sliders
	falloffSlider := widget.NewSlider(0.5, 4.0)
	falloffSlider.Step = 0.05
	falloffSlider.Value = falloff
	falloffSlider.OnChanged = func(v float64) {
		falloff = v
		falloffLabel.SetText(fmt.Sprintf("Falloff: %.2f", falloff))
		triggerUpdate()
	}

	falloffWeightSlider := widget.NewSlider(0.0, 1.0)
	falloffWeightSlider.Step = 0.01
	falloffWeightSlider.Value = falloffWeight
	falloffWeightSlider.OnChanged = func(v float64) {
		falloffWeight = v
		falloffWeightLabel.SetText(fmt.Sprintf("Falloff Weight: %.2f", falloffWeight))
		triggerUpdate()
	}

	// Sea level
	seaLevelSlider := widget.NewSlider(0.0, 1.0)
	seaLevelSlider.Step = 0.01
	seaLevelSlider.Value = seaLevel
	seaLevelSlider.OnChanged = func(v float64) {
		seaLevel = v
		seaLevelLabel.SetText(fmt.Sprintf("Sea Level: %.2f", seaLevel))
		triggerUpdate()
	}

	// Min distance for POIs
	minDistanceSlider := widget.NewSlider(1, 50)
	minDistanceSlider.Step = 1
	minDistanceSlider.Value = float64(minDistance)
	minDistanceSlider.OnChanged = func(v float64) {
		minDistance = int64(v)
		minDistanceLabel.SetText(fmt.Sprintf("Min. Distance: %d", minDistance))
		triggerUpdate()
	}

	// Flow sliders
	flowScaleSlider := widget.NewSlider(0.0, 0.02)
	flowScaleSlider.Step = 0.0005
	flowScaleSlider.Value = flowScale
	flowScaleSlider.OnChanged = func(v float64) {
		flowScale = v
		flowScaleLabel.SetText(fmt.Sprintf("Flow Scale: %.4f", flowScale))
		triggerUpdate()
	}

	flowStrengthSlider := widget.NewSlider(0, 60)
	flowStrengthSlider.Step = 1
	flowStrengthSlider.Value = flowStrength
	flowStrengthSlider.OnChanged = func(v float64) {
		flowStrength = v
		flowStrengthLabel.SetText(fmt.Sprintf("Flow Strength: %.2f", flowStrength))
		triggerUpdate()
	}

	// Save button (capture image under mutex first)
	saveButton := widget.NewButton("Save PNG", func() {
		mutex.Lock()
		toSave := img
		mutex.Unlock()

		tempFilename := fmt.Sprintf("world_%d.png", time.Now().Unix())
		f, err := os.Create(tempFilename)
		if err != nil {
			fmt.Println("save create error:", err)
			return
		}
		defer f.Close()

		if err := png.Encode(f, toSave); err != nil {
			fmt.Println("png encode error:", err)
		}
	})

	controls := container.NewVBox(
		widget.NewLabel("Use the sliders below to adjust the world."),
		seedLabel, seedSlider, randomSeedBtn,
		scaleLabel, scaleSlider,
		octavesLabel, octavesSlider,
		persistenceLabel, persistenceSlider,
		lacunarityLabel, lacunaritySlider,
		continentFreqLabel, continentFreqSlider,
		continentOctavesLabel, continentOctavesSlider,
		continentWeightLabel, continentWeightSlider,
		falloffLabel, falloffSlider,
		falloffWeightLabel, falloffWeightSlider,
		seaLevelLabel, seaLevelSlider,
		minDistanceLabel, minDistanceSlider,
		flowScaleLabel, flowScaleSlider,
		flowStrengthLabel, flowStrengthSlider,
		saveButton,
	)

	scrollableControls := container.NewScroll(controls)

	split := container.NewHSplit(
		imageCanvas,
		scrollableControls,
	)
	split.Offset = 0.75 // Adjust the initial split ratio

	myWindow.SetContent(split)

	// initial render
	triggerUpdate()
	myWindow.ShowAndRun()
}