package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"os"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"
	"github.com/joho/godotenv"
)

type BenchmarkResult struct {
	ModelName       string  `json:"model_name"`
	Timestamp       int64   `json:"timestamp"`
	Duration        float64 `json:"duration"`
	TokensPerSecond float64 `json:"tokens_per_second"`
	EvalCount       int     `json:"eval_count"`
	EvalDuration    int64   `json:"eval_duration"`
	Iterations      int     `json:"iterations"`
}

type OllamaRequest struct {
	ModelName string `json:"model"`
	Prompt    string `json:"prompt"`
}

type OllamaResponse struct {
	Model        string `json:"model"`
	CreatedAt    string `json:"created_at"`
	Response     string `json:"response"`
	Done         bool   `json:"done"`
	EvalCount    int    `json:"eval_count"`
	EvalDuration int64  `json:"eval_duration"`
}

// Models supported
var MODELS = []string{
	"llama3",
	"phi3",
	"mistral",
	"mixtral:8x22b",
	"command-r",
	"command-r-plus",
	"dolphin-llama3",
	"dolphin-mixtral:8x22b",
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
	}

	flag.Usage = func() {
		fmt.Println("Usage: ollamark [options]")
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println("Examples:")
		fmt.Println("  For Ollamark GUI mode:")
		fmt.Println("      ollamark (no flags)")
		fmt.Println("  For Ollamark CLI mode:")
		fmt.Println("      ollamark -m llama3 -i 10")
		fmt.Println("      ollamark -m phi3")
		fmt.Println("      ollamark -m phi3 -s -o http://localhost:11434/api/generate")
	}

	// Parse command-line arguments (Ollamark CLI)
	modelPtr := flag.String("m", "llama3", "Model name to benchmark")
	submitPtr := flag.Bool("s", false, "Submit benchmark results to Ollamark (default false)")
	ollamaPtr := flag.String("o", "http://localhost:11434/api/generate", "Ollama API endpoint")
	iterationsPtr := flag.Int("i", 2, "Number of benchmark iterations (Min 2, Max 20)")
	flag.Parse()

	// Check if CLI arguments are provided
	if flag.NFlag() > 0 {

		if *modelPtr == "" || *ollamaPtr == "" {
			flag.Usage()
			os.Exit(1)
		}

		if flag.NArg() > 0 {
			flag.Usage()
			os.Exit(1)
		}

		if (*iterationsPtr < 2) || (*iterationsPtr > 20) {
			flag.Usage()
			os.Exit(1)
		}

		// Run ollamark in CLI mode
		runBenchmarkCLI(*modelPtr, *submitPtr, *ollamaPtr, *iterationsPtr)
		return
	}

	// Create a new Fyne app
	a := app.NewWithID("Ollamark")
	a.Settings().SetTheme(theme.DarkTheme())
	fyne.CurrentApp().Settings().SetTheme(fyne.CurrentApp().Settings().Theme())
	w := a.NewWindow("Ollamark - Ollama Benchmark")

	// set window size
	w.Resize(fyne.NewSize(300, 200))
	w.CenterOnScreen()

	// create a logo
	logo := canvas.NewImageFromFile("logo.svg")
	logo.FillMode = canvas.ImageFillContain // Use 'Contain' to ensure the image fits well
	logo.SetMinSize(fyne.NewSize(100, 100))

	// Load the SVG icon
	icon, err := fyne.LoadResourceFromPath("logo.svg")
	if err != nil {
		// Handle the error if the icon file cannot be loaded
		fmt.Println("Failed to load icon:", err)
	} else {
		// Set the application icon
		a.SetIcon(icon)
	}

	// create an api entry field
	apiEntry := widget.NewEntry()
	apiEntry.SetText("http://localhost:11434/api/generate")

	// create a title label
	titleLabel := widget.NewLabel("Ollama API Endpoint")
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	title2Label := widget.NewLabel("Select a model to benchmark")
	title2Label.TextStyle = fyne.TextStyle{Bold: true}

	modelSelect := widget.NewSelect(MODELS, func(value string) {
		// do nothing
	})
	modelSelect.SetSelected("llama3")

	resultLabel := widget.NewLabel("")
	resultLabel.Alignment = fyne.TextAlignCenter
	resultLabel.Hide()

	// Custom text field for tokens per second
	tokensPerSecondText := canvas.NewText("", color.White)
	tokensPerSecondText.TextStyle.Bold = true
	tokensPerSecondText.TextSize = 38 // Larger text size
	tokensPerSecondText.Alignment = fyne.TextAlignCenter
	tokensPerSecondText.Hide()

	tpsText := canvas.NewText("", color.White)
	tpsText.TextStyle.Bold = true
	tpsText.TextSize = 16 // Larger text size
	tpsText.Alignment = fyne.TextAlignCenter
	tpsText.Hide()

	iterationsSlider := widget.NewSlider(2, 20)
	iterationsSlider.SetValue(2)
	iterationsSlider.Step = 1

	iterationsLabel := widget.NewLabel("Iterations: 2")
	iterationsSlider.OnChanged = func(value float64) {
		iterationsLabel.SetText(fmt.Sprintf("Iterations: %d", int(value)))
	}

	// create a progress bar
	progressBar := widget.NewProgressBarInfinite()
	progressBar.Hide()

	gifURI := storage.NewFileURI("loader.gif")
	gif, err := xwidget.NewAnimatedGif(gifURI)
	if err != nil {
		fmt.Println("Error loading gif:", err)
	} else {
		gif.Start()
		gif.Show()
	}

	var benchmarkResult *BenchmarkResult
	var submitButton *widget.Button

	benchmarkButton := widget.NewButton("Benchmark", nil)
	benchmarkButton.OnTapped = func() {
		benchmarkButton.SetText("Benchmarking...")
		benchmarkButton.Disable()
		submitButton.Disable()

		resultLabel.Show()
		resultLabel.SetText("Benchmarks starting...")
		resultLabel.Refresh()

		tokensPerSecondText.Hide()
		tpsText.Hide()

		go func() {
			progressBar.Show()
			progressBar.Refresh()

			// get api url and model name from entry fields
			apiURL := apiEntry.Text
			modelName := modelSelect.Selected
			iterations := int(iterationsSlider.Value)

			var totalTokensPerSecond float64

			for i := 0; i < iterations; i++ {
				requestBody := OllamaRequest{
					ModelName: modelName,
					Prompt:    "Tell me about Llamas in 500 words.",
				}

				jsonData, _ := json.Marshal(requestBody)
				resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
				if err != nil {
					resultLabel.SetText("Error: " + err.Error())
					benchmarkButton.SetText("Benchmark")
					benchmarkButton.Enable()
					progressBar.Hide()
					progressBar.Refresh()
					gif.Hide()
					return
				}
				defer resp.Body.Close()

				// start := time.Now()

				var response OllamaResponse
				var responseText string
				decoder := json.NewDecoder(resp.Body)

				resultLabel.SetText(fmt.Sprintf("Benchmark #%d in progress...", i+1))
				resultLabel.Refresh()

				for {
					err := decoder.Decode(&response)
					if err == io.EOF {
						break
					}
					if err != nil {
						resultLabel.SetText("Error: " + err.Error())
						progressBar.Hide()
						progressBar.Refresh()
						benchmarkButton.SetText("Benchmark")
						benchmarkButton.Enable()
						return
					}

					responseText += response.Response
					progressBar.Refresh()
				}

				// duration := time.Since(start).Seconds()
				tokensPerSecond := float64(response.EvalCount) / (float64(response.EvalDuration) / 1e9)

				totalTokensPerSecond += tokensPerSecond
			}

			avgTokensPerSecond := totalTokensPerSecond / float64(iterations)

			benchmarkResult = &BenchmarkResult{
				ModelName:       modelName,
				Timestamp:       time.Now().Unix(),
				TokensPerSecond: avgTokensPerSecond,
				Iterations:      iterations,
			}

			resultLabel.SetText(fmt.Sprintf("Benchmark completed for %s\nAverage Tokens per second: %.2f\nBenchmarked with %d iterations", modelName, avgTokensPerSecond, iterations))
			resultLabel.Alignment = fyne.TextAlignCenter
			resultLabel.Refresh()

			// update custom text
			tokensPerSecondText.Text = fmt.Sprintf("%.2f", avgTokensPerSecond) // Update the custom text
			tokensPerSecondText.Show()
			tpsText.Text = "Tokens per second"
			tokensPerSecondText.Refresh()
			tpsText.Refresh() // Refresh to update the display
			tpsText.Show()

			progressBar.Hide()
			gif.Hide()
			progressBar.Refresh() // Refresh after hiding the ProgressBar
			benchmarkButton.SetText("Benchmark Again")
			benchmarkButton.Enable()
			submitButton.Show()
			submitButton.Enable()
		}()
	}

	submitButton = widget.NewButton("Submit Benchmark", nil)
	submitButton.OnTapped = func() {
		if benchmarkResult != nil {
			apiEndpoint := os.Getenv("SUBMISSION_API_ENDPOINT")
			apiKey := os.Getenv("API_KEY")

			jsonData, _ := json.Marshal(benchmarkResult)
			req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(jsonData))
			if err != nil {
				resultLabel.SetText("Error creating request: " + err.Error())
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+apiKey)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				resultLabel.SetText("Error submitting benchmark: " + err.Error())
			} else {
				resp.Body.Close()
				resultLabel.SetText("Benchmark submitted successfully!")
			}
		}
	}
	submitButton.Hide()

	content := container.NewVBox(
		logo,
		titleLabel,
		apiEntry,
		title2Label,
		modelSelect,
		iterationsLabel,
		iterationsSlider,
		gif,
		widget.NewSeparator(),
		tokensPerSecondText,
		tpsText,
		resultLabel,
		progressBar,
		widget.NewSeparator(),
		benchmarkButton,
		submitButton,
	)

	w.SetContent(content)
	w.ShowAndRun()
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func runBenchmarkCLI(modelName string, submit bool, ollamaAPI string, iterations int) {
	// Get Ollama API URL from environment variable
	ollamaAPIURL := ollamaAPI

	var totalTokensPerSecond float64

	// modelName needs to match a model name in MODELS
	if !contains(MODELS, modelName) {
		fmt.Println("Model not supported. Please use a supported model from the list:", MODELS)
		return
	}

	for i := 0; i < iterations; i++ {
		requestBody := OllamaRequest{
			ModelName: modelName,
			Prompt:    "Tell me about Llamas in 500 words.",
		}

		jsonData, _ := json.Marshal(requestBody)
		resp, err := http.Post(ollamaAPIURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()

		// start := time.Now()

		var response OllamaResponse
		var responseText string
		decoder := json.NewDecoder(resp.Body)

		fmt.Printf("Benchmarking iteration %d in progress..", i+1)
		progressTicker := time.NewTicker(500 * time.Millisecond)
		defer progressTicker.Stop()

		done := make(chan bool)
		go func() {
			for {
				select {
				case <-progressTicker.C:
					fmt.Print(".")
				case <-done:
					fmt.Println()
					return
				}
			}
		}()

		for {
			err := decoder.Decode(&response)
			if err == io.EOF {
				done <- true
				break
			}
			if err != nil {
				fmt.Println("\nError:", err)
				done <- true
				return
			}

			responseText += response.Response
		}

		// duration := time.Since(start).Seconds()
		tokensPerSecond := float64(response.EvalCount) / (float64(response.EvalDuration) / 1e9)

		totalTokensPerSecond += tokensPerSecond

	}

	avgTokensPerSecond := totalTokensPerSecond / float64(iterations)

	fmt.Printf("\nBenchmark completed for %s\n", modelName)
	fmt.Printf("Average Tokens per second: %.2f\n", avgTokensPerSecond)

	benchmarkResult := &BenchmarkResult{
		ModelName:       modelName,
		Timestamp:       time.Now().Unix(),
		TokensPerSecond: avgTokensPerSecond,
		Iterations:      iterations,
	}

	if submit {
		// Get Submission API URL from environment variable
		submissionAPIURL := os.Getenv("SUBMISSION_API_ENDPOINT")
		apiKey := os.Getenv("API_KEY")

		jsonData, _ := json.Marshal(benchmarkResult)
		req, err := http.NewRequest("POST", submissionAPIURL, bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Println("Error creating request:", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Error submitting benchmark:", err)
		} else {
			resp.Body.Close()
			fmt.Println("Benchmark submitted successfully!")
		}
	} else {
		fmt.Println("Benchmark results not submitted.")
	}
}
