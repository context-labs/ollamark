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

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
	}

	// Parse command-line arguments (Ollamark CLI)
	modelPtr := flag.String("m", "llama3", "Model name to benchmark")
	submitPtr := flag.Bool("s", false, "Submit benchmark results")
	ollamaPtr := flag.String("o", "http://localhost:11434/api/generate", "Ollama API endpoint")
	flag.Parse()

	// Check if CLI arguments are provided
	if flag.NFlag() > 0 {
		// Run ollamark in CLI mode
		runBenchmarkCLI(*modelPtr, *submitPtr, *ollamaPtr)
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

	modelSelect := widget.NewSelect([]string{
		"llama3",
		"llama2",
		"mistral",
		"mixtral:8x22b",
		"command-r",
		"command-r-plus",
		"tinydolphin",
		"dolphinphi",
	}, func(value string) {
		// do nothing
	})
	modelSelect.SetSelected("llama3")

	resultLabel := widget.NewLabel("")

	// Custom text field for tokens per second
	tokensPerSecondText := canvas.NewText("", color.White)
	tokensPerSecondText.TextStyle.Bold = true
	tokensPerSecondText.TextSize = 38 // Larger text size
	tokensPerSecondText.Alignment = fyne.TextAlignCenter

	tpsText := canvas.NewText("", color.White)
	tpsText.TextStyle.Bold = true
	tpsText.TextSize = 16 // Larger text size
	tpsText.Alignment = fyne.TextAlignCenter

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
		go func() {
			progressBar.Show()
			progressBar.Refresh()

			// get api url and model name from entry fields
			apiURL := apiEntry.Text
			modelName := modelSelect.Selected

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

			start := time.Now()

			var response OllamaResponse
			var responseText string
			decoder := json.NewDecoder(resp.Body)
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

			duration := time.Since(start).Seconds()
			tokensPerSecond := float64(response.EvalCount) / (float64(response.EvalDuration) / 1e9)

			benchmarkResult = &BenchmarkResult{
				ModelName:       modelName,
				Timestamp:       time.Now().Unix(),
				Duration:        duration,
				TokensPerSecond: tokensPerSecond,
				EvalCount:       response.EvalCount,
				EvalDuration:    response.EvalDuration,
			}

			resultLabel.SetText(fmt.Sprintf("Benchmark completed for %s\nDuration: %.2f seconds", modelName, duration))
			resultLabel.Alignment = fyne.TextAlignCenter
			resultLabel.Refresh()

			// update custom text
			tokensPerSecondText.Text = fmt.Sprintf("%.2f", tokensPerSecond) // Update the custom text
			tpsText.Text = "Tokens per second"
			tokensPerSecondText.Refresh()
			tpsText.Refresh() // Refresh to update the display

			progressBar.Hide()
			gif.Hide()
			progressBar.Refresh() // Refresh after hiding the ProgressBar
			benchmarkButton.SetText("Benchmark Again")
			benchmarkButton.Enable()
			submitButton.Show()
		}()
	}

	submitButton = widget.NewButton("Submit Benchmark", nil)
	submitButton.OnTapped = func() {
		if benchmarkResult != nil {
			apiEndpoint := os.Getenv("API_ENDPOINT")
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
		benchmarkButton,
		submitButton,
		gif,
		widget.NewSeparator(),
		tokensPerSecondText,
		tpsText,
		resultLabel,
		progressBar,
	)

	w.SetContent(content)
	w.ShowAndRun()
}

func runBenchmarkCLI(modelName string, submit bool, ollamaAPI string) {
	// Get Ollama API URL from environment variable
	ollamaAPIURL := ollamaAPI

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

	start := time.Now()

	var response OllamaResponse
	var responseText string
	decoder := json.NewDecoder(resp.Body)

	fmt.Print("Benchmarking in progress..")
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

	duration := time.Since(start).Seconds()
	tokensPerSecond := float64(response.EvalCount) / (float64(response.EvalDuration) / 1e9)

	benchmarkResult := &BenchmarkResult{
		ModelName:       modelName,
		Timestamp:       time.Now().Unix(),
		Duration:        duration,
		TokensPerSecond: tokensPerSecond,
		EvalCount:       response.EvalCount,
		EvalDuration:    response.EvalDuration,
	}

	fmt.Printf("\nBenchmark completed for %s\n", modelName)
	fmt.Printf("Duration: %.2f seconds\n", duration)
	fmt.Printf("Tokens per second: %.2f\n", tokensPerSecond)

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
