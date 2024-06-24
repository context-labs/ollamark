// Ollamark By Carsen Klock 2024 under the MIT license
// https://github.com/context-labs/ollamark
// https://ollamark.com
// Ollamark Client

package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"image/color"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/mem"
)

type BenchmarkResult struct {
	ModelName       string              `json:"model_name"`
	Timestamp       int64               `json:"timestamp"`
	Duration        float64             `json:"duration"`
	TokensPerSecond float64             `json:"tokens_per_second"`
	EvalCount       int                 `json:"eval_count"`
	EvalDuration    int64               `json:"eval_duration"`
	Iterations      int                 `json:"iterations"`
	SysInfo         *SysInfo            `json:"sys_info"`
	GPUInfo         *GPUInfo            `json:"gpu_info"`
	OllamaVersion   string              `json:"ollama_version"`
	ClientType      string              `json:"client_type"`
	ClientVersion   string              `json:"client_version"`
	IP              string              `json:"ip"`
	ProofOfWork     ProofOfWorkSolution `json:"proof_of_work"`
}

type OllamaRequest struct {
	ModelName string `json:"model"`
	Prompt    string `json:"prompt"`
}

type ModelRequest struct {
	Name string `json:"name"`
}

type OllamaResponse struct {
	Model        string `json:"model"`
	CreatedAt    string `json:"created_at"`
	Response     string `json:"response"`
	Done         bool   `json:"done"`
	EvalCount    int    `json:"eval_count"`
	EvalDuration int64  `json:"eval_duration"`
}

type SysInfo struct {
	OS      string `json:"os"`
	Arch    string `json:"arch"`
	Version string `json:"version"`
	Kernel  string `json:"kernel"`
	CPU     string `json:"cpu"`
	CPUName string `json:"cpu_name"`
	Memory  string `json:"memory"`
}

type GPUInfo struct {
	Name          string `json:"name"`
	Vendor        string `json:"vendor"`
	Memory        string `json:"memory"`
	DriverVersion string `json:"driver_version"`
	Count         int    `json:"count"`
}

var (
	globalModels  []ModelInfo
	apiEndpoint   string
	clientVersion = "0.0.1"
)

// ProofOfWorkChallenge represents a proof-of-work challenge
type ProofOfWorkChallenge struct {
	Challenge  string `json:"challenge"`
	Difficulty int    `json:"difficulty"`
	Timestamp  int64  `json:"timestamp"`
}

// ProofOfWorkSolution represents a solution to a proof-of-work challenge
type ProofOfWorkSolution struct {
	Challenge  string `json:"challenge"`
	Nonce      string `json:"nonce"`
	Timestamp  int64  `json:"timestamp"`
	Difficulty int    `json:"difficulty"`
}

// requestProofOfWorkChallenge requests a new proof-of-work challenge from the server
func requestProofOfWorkChallenge(apiEndpoint string) (ProofOfWorkChallenge, error) {
	resp, err := http.Get(apiEndpoint + "/api/pow-challenge")
	if err != nil {
		return ProofOfWorkChallenge{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ProofOfWorkChallenge{}, err
	}

	var challenge ProofOfWorkChallenge
	if err := json.Unmarshal(body, &challenge); err != nil {
		return ProofOfWorkChallenge{}, err
	}

	return challenge, nil
}

// solveProofOfWork solves the proof-of-work challenge
func solveProofOfWork(challenge ProofOfWorkChallenge) (string, error) {
	prefix := strings.Repeat("0", challenge.Difficulty)
	for i := 0; ; i++ {
		nonce := strconv.Itoa(i)
		hash := sha256.Sum256([]byte(challenge.Challenge + nonce))
		if strings.HasPrefix(hex.EncodeToString(hash[:]), prefix) {
			return nonce, nil
		}
	}
}

type ModelInfo struct {
	Name         string `json:"name"`
	Parameters   string `json:"parameters"`
	Quantization string `json:"quantization"`
}

func fetchModels() ([]ModelInfo, error) {
	mainURL := os.Getenv("OLLAMARK_API")
	resp, err := http.Get(mainURL + "/api/model-list")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the raw JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON response
	var result struct {
		Models []ModelInfo `json:"models"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Models, nil
}

func initModels() error {
	models, err := fetchModels()
	if err != nil {
		return err
	}
	globalModels = models
	return nil
}

func LoadPublicKey() (*rsa.PublicKey, error) {
	publicKeyData := os.Getenv("PUBLIC_KEY")
	block, _ := pem.Decode([]byte(publicKeyData))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block containing the public key")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to RSA public key")
	}

	return rsaPublicKey, nil
}

func EncryptData(publicKey *rsa.PublicKey, data []byte) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, data, nil)
}

// Generate a random AES key
func generateAESKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// Encrypt data using AES-GCM
func encryptAESGCM(key, plaintext []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext := aesGCM.Seal(nil, nonce, plaintext, nil)
	return nonce, ciphertext, nil
}

// Encrypt the AES key using RSA
func encryptRSA(publicKey *rsa.PublicKey, data []byte) ([]byte, error) {
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, data, nil)
}

// Generate a random UUID
func generateUUID() string {
	return uuid.New().String()
}

// Sign the UUID with HMAC-SHA256
func signUUID(uuid string, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(uuid))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func getCPUName() string {
	// get windows cpu name
	if runtime.GOOS == "windows" {
		cmd := exec.Command("wmic", "cpu", "get", "name")
		output, err := cmd.Output()
		if err != nil {
			return "Unknown"
		}
		lines := strings.Split(string(output), "\n")
		if len(lines) > 1 {
			return strings.TrimSpace(lines[1])
		}
		return "Unknown"
	}

	// get linux cpu name
	if runtime.GOOS == "linux" {
		cmd := exec.Command("lshw", "-C", "cpu")
		output, err := cmd.Output()
		if err != nil {
			return "Unknown"
		}
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "product:") {
				return strings.TrimSpace(strings.Split(line, ":")[1])
			}
		}
		return "Unknown"
	}

	return "Unknown"
}

func getKernelVersion() (string, error) {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("wmic", "os", "get", "Version", "/value")
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Version=") {
				return strings.TrimSpace(strings.Split(line, "=")[1]), nil
			}
		}
		return "", fmt.Errorf("failed to parse Windows version")
	}

	cmd := exec.Command("uname", "-r")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getSysInfo() (*SysInfo, error) {
	v, _ := mem.VirtualMemory()
	// s, _ := mem.SwapMemory()

	totalMemory := v.Total / 1024 / 1024 / 1024
	// usedMemory := v.Used
	// availableMemory := v.Available
	// swapTotal := s.Total
	// swapUsed := s.Used

	sysInfo := &SysInfo{}
	sysInfo.OS = runtime.GOOS
	sysInfo.Arch = runtime.GOARCH
	sysInfo.Version = "0.0.1"
	kernelVersion, err := getKernelVersion()
	if err != nil {
		return nil, err
	}
	sysInfo.Kernel = kernelVersion
	sysInfo.CPU = strconv.Itoa(runtime.NumCPU())
	// get CPU Name for Windows and Linux

	sysInfo.CPUName = getCPUName()

	sysInfo.Memory = strconv.Itoa(int(totalMemory)) + " GB"

	// Get system information if macOS (darwin) and aarch64 (arm64) then get the info with apple silicon only command: TODO (Test)
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		cmd := exec.Command("system_profiler", "SPHardwareDataType")
		output, err := cmd.Output()
		if err != nil {
			return nil, err
		}
		// Extract the CPU information from the output
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Chip:") {
				sysInfo.CPUName = strings.TrimSpace(strings.Split(line, ":")[1])
				break
			}
		}
	}

	return sysInfo, nil
}

func getMacGPUInfo() (*GPUInfo, error) {
	cmd := exec.Command("system_profiler", "SPDisplaysDataType")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	gpuInfo := &GPUInfo{}
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Chipset Model:") {
			gpuInfo.Name = strings.TrimSpace(strings.Split(line, ":")[1])
			gpuInfo.Vendor = "Apple"
			break
		}
	}

	// If we couldn't find GPU info, it's likely integrated with the CPU
	if gpuInfo.Name == "" {
		cpuCmd := exec.Command("system_profiler", "SPHardwareDataType")
		cpuOutput, err := cpuCmd.Output()
		if err != nil {
			return nil, err
		}
		cpuLines := strings.Split(string(cpuOutput), "\n")
		for _, line := range cpuLines {
			if strings.Contains(line, "Chip:") {
				gpuInfo.Name = strings.TrimSpace(strings.Split(line, ":")[1]) + " GPU"
				gpuInfo.Vendor = "Apple"
				break
			}
		}
	}

	// Memory information isn't easily available for integrated GPUs
	gpuInfo.Memory = "Shared"
	gpuInfo.DriverVersion = "N/A"
	gpuInfo.Count = 1

	return gpuInfo, nil
}

func getGPUInfo() (*GPUInfo, error) {
	// First, attempt to use nvidia-smi to fetch Nvidia GPU info
	nvidiaGPU, err := getNvidiaGPUInfo()
	if err == nil {
		return nvidiaGPU, nil
	}

	// If Nvidia GPU info fetching fails, attempt to fetch AMD GPU info
	amdGPU, err := getAMDGPUInfo()
	if err == nil {
		return amdGPU, nil
	}

	// Check if we're on macOS (darwin) and arm64 architecture
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		return getMacGPUInfo()
	}

	// If both methods fail, return the last error
	return nil, err
}

func getNvidiaGPUInfo() (*GPUInfo, error) {
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,memory.total,driver_version", "--format=csv,noheader")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	outputStr := strings.TrimSpace(string(output))
	lines := strings.Split(outputStr, "\n")[0] // Assuming single GPU
	fields := strings.Split(lines, ",")

	if len(fields) < 2 {
		return nil, fmt.Errorf("failed to parse Nvidia GPU information")
	}

	return &GPUInfo{
		Name:          strings.TrimSpace(fields[0]),
		Vendor:        "NVIDIA",
		Memory:        strings.TrimSpace(fields[1]),
		DriverVersion: strings.TrimSpace(fields[2]),
		Count:         len(lines),
	}, nil
}

func getAMDGPUInfo() (*GPUInfo, error) {
	switch runtime.GOOS {
	case "windows":
		return getAMDGPUInfoWindows()
	case "linux":
		return getAMDGPUInfoLinux()
	case "darwin":
		return nil, fmt.Errorf("macOS, Skipping AMD GPU info")
	default:
		return nil, fmt.Errorf("AMD GPU unsupported operating system")
	}
}

func getAMDGPUInfoWindows() (*GPUInfo, error) {
	cmd := exec.Command("wmic", "path", "win32_VideoController", "get", "Name,DriverVersion", "/format:list")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to execute wmic command: %v", err)
	}

	outputStr := string(output)
	return parseWMICOutput(outputStr)
}

func parseWMICOutput(output string) (*GPUInfo, error) {
	lines := strings.Split(output, "\n")
	info := GPUInfo{}
	gpuNames := make(map[string]bool) // To track unique GPU names

	for _, line := range lines {
		if strings.HasPrefix(line, "Name=") {
			name := strings.TrimSpace(strings.Split(line, "=")[1])
			// Skip integrated and virtual GPUs
			if strings.Contains(name, "Integrated") || strings.Contains(name, "Display Adapter") || strings.Contains(name, "AMD Radeon(TM) Graphics") {
				continue
			}
			if !gpuNames[name] {
				gpuNames[name] = true
				info.Name = name
				info.Vendor = "AMD" // Assuming AMD if we are parsing this on an AMD system check
				info.Count++
			}
		} else if strings.HasPrefix(line, "DriverVersion=") {
			info.DriverVersion = strings.TrimSpace(strings.Split(line, "=")[1])
			info.Memory = "Unknown" // Placeholder for memory, as WMIC does not provide it directly
		}
	}

	if info.Name == "" {
		return nil, fmt.Errorf("no dedicated AMD GPUs found")
	}

	return &info, nil
}

func getAMDGPUInfoLinux() (*GPUInfo, error) {
	cmd := exec.Command("lshw", "-C", "display")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	outputStr := string(output)
	// Example of parsing, adjust according to actual output
	if strings.Contains(outputStr, "Radeon") || strings.Contains(outputStr, "AMD") {
		name := extractField(outputStr, "product")
		// vendor := "AMD"
		memory := extractField(outputStr, "size")

		return &GPUInfo{
			Name: name,
			// Vendor: vendor,
			Memory: memory,
		}, nil
	}

	return nil, fmt.Errorf("no AMD GPU detected")
}

func getIPAddress() string {
	resp, err := http.Get("https://icanhazip.com")
	if err != nil {
		return "Unknown"
	}
	defer resp.Body.Close()
	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "Unknown"
	}
	return strings.TrimSpace(string(ip))
}

func getOllamaVersion() string {
	cmd := exec.Command("ollama", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "Unknown"
	}

	// remove "ollama version is " from the output
	return strings.TrimSpace(strings.Split(string(output), "ollama version is ")[1])
}

func extractField(data, fieldName string) string {
	// Simple parsing logic, needs to be adjusted based on actual output
	start := strings.Index(data, fieldName+":")
	if start == -1 {
		return ""
	}
	start += len(fieldName) + 1
	end := strings.Index(data[start:], "\n")
	if end == -1 {
		return data[start:]
	}
	return strings.TrimSpace(data[start : start+end])
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
	}

	fmt.Println("Loading Ollamark...")

	fmt.Println("Checking Ollama Version...")
	ollamaVersion := getOllamaVersion()
	if ollamaVersion == "Unknown" {
		fmt.Println("Ollama not found, please install Ollama from https://ollama.com/download to Ollamark 😎")
		return
	}
	fmt.Println("Ollama Version:", ollamaVersion)

	err = initModels()
	if err != nil {
		fmt.Println("Failed to initialize models:", err)
		return
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
		fmt.Println("      ollamark -m phi3 -s")
		fmt.Println("      ollamark -m phi3 -s -o http://localhost:11434/api/generate")
	}

	// Parse command-line arguments (Ollamark CLI)
	modelPtr := flag.String("m", "llama3", "Model name to benchmark (default: llama3)")
	submitPtr := flag.Bool("s", false, "Submit benchmark results to Ollamark.com (default false)")
	ollamaPtr := flag.String("o", "http://localhost:11434", "Ollama API endpoint (default http://localhost:11434)")
	iterationsPtr := flag.Int("i", 2, "Number of benchmark iterations (Min 2, Max 20)")
	flag.Parse()

	// Set the global API endpoint
	apiEndpoint = *ollamaPtr

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
		runBenchmarkCLI(*modelPtr, *submitPtr, apiEndpoint, *iterationsPtr)
		return
	}

	// Create a new Fyne app
	a := app.NewWithID("Ollamark")
	a.Settings().SetTheme(theme.DarkTheme())
	fyne.CurrentApp().Settings().SetTheme(fyne.CurrentApp().Settings().Theme())
	w := a.NewWindow("Ollamark - Ollama Benchmark")

	// set window size
	w.Resize(fyne.NewSize(400, 300))
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

	sysinfo, _ := getSysInfo()
	gpuinfo, _ := getGPUInfo()
	ollamaVersion = getOllamaVersion()

	// create an api entry field
	apiEntry := widget.NewEntry()
	apiEntry.SetText(apiEndpoint)

	// create a title label
	titleLabel := widget.NewLabel("Ollama API Endpoint")
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	title2Label := widget.NewLabel("Select a model to benchmark")
	title2Label.TextStyle = fyne.TextStyle{Bold: true}

	// Create a slice of model names for the dropdown
	modelNames := make([]string, len(globalModels))
	for i, model := range globalModels {
		modelNames[i] = model.Name
	}

	// Create the select widget with model names
	modelSelect := widget.NewSelect(modelNames, func(value string) {
		// You can add logic here if needed when a model is selected
	})

	// Set the default selected model
	// Find the index of "llama3" in the modelNames slice
	defaultIndex := 0
	for i, name := range modelNames {
		if name == "llama3" {
			defaultIndex = i
			break
		}
	}
	modelSelect.SetSelected(modelNames[defaultIndex])

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

	sysText := widget.NewLabel("")
	sysText.Hide()

	gpuText := widget.NewLabel("")
	gpuText.Hide()

	ollamaVersionText := widget.NewLabel("")
	ollamaVersionText.Hide()

	iterationsSlider := widget.NewSlider(2, 20)
	iterationsSlider.SetValue(2)
	iterationsSlider.Step = 1

	iterationsLabel := widget.NewLabel("Iterations: 2")
	iterationsSlider.OnChanged = func(value float64) {
		iterationsLabel.SetText(fmt.Sprintf("Iterations: %d", int(value)))
	}

	sysText.SetText(fmt.Sprintf("CPU: %s\nMemory: %s\nOS: %s\nKernel: %s", sysinfo.CPUName, sysinfo.Memory, sysinfo.OS, sysinfo.Kernel))
	sysText.Show()
	sysText.Refresh()

	// if gpu Info is available, show it
	if gpuinfo != nil {
		gpuText.SetText(fmt.Sprintf("GPU Name: %s\nDriver Version: %s", gpuinfo.Name, gpuinfo.DriverVersion))
		gpuText.Show()
		gpuText.Refresh()
	}

	// set ollama version text make version bold
	ollamaVersionText.SetText(fmt.Sprintf("Ollama Version: %s", ollamaVersion))
	ollamaVersionText.Show()
	ollamaVersionText.Refresh()

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
	var linkButton *widget.Button

	benchmarkButton := widget.NewButton("Benchmark", nil)
	benchmarkButton.OnTapped = func() {
		linkButton.Hide()
		benchmarkButton.SetText("Benchmarking...")
		benchmarkButton.Disable()
		submitButton.Disable()

		resultLabel.Show()
		resultLabel.SetText("Benchmarks starting...")
		resultLabel.Refresh()

		tokensPerSecondText.Hide()
		tpsText.Hide()
		// sysText.Hide()
		// gpuText.Hide()

		go func() {
			progressBar.Show()
			progressBar.Refresh()

			// get api url and model name from entry fields
			apiURL := apiEntry.Text
			modelName := modelSelect.Selected
			iterations := int(iterationsSlider.Value)

			modelRequest := ModelRequest{
				Name: modelName,
			}
			jsonData, _ := json.Marshal(modelRequest)
			fullURL := apiEndpoint + "/api/pull"
			resultLabel.SetText("Pulling model " + modelName + ", Please wait...")
			resultLabel.Refresh()
			resp, err := http.Post(fullURL, "application/json", bytes.NewBuffer(jsonData))
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

			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != http.StatusOK {
				resultLabel.SetText(fmt.Sprintf("Error pulling model: %s", body))
				benchmarkButton.SetText("Benchmark")
				benchmarkButton.Enable()
				progressBar.Hide()
				progressBar.Refresh()
				gif.Hide()
				return
			}

			// fmt.Println("Model pull response:", string(body)) // Debug print
			resultLabel.SetText("Model pulled successfully")
			resultLabel.Refresh()
			resultLabel.SetText("Benchmarking...")
			resultLabel.Refresh()

			var totalTokensPerSecond float64
			var evalCount int
			var evalDuration float64

			start := time.Now()

			for i := 0; i < iterations; i++ {
				requestBody := OllamaRequest{
					ModelName: modelName,
					Prompt:    "Tell me about Llamas in 500 words.",
				}

				jsonData, _ := json.Marshal(requestBody)
				resp, err := http.Post(apiURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
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
				evalCount = response.EvalCount
				evalDuration = float64(response.EvalDuration) / 1e9
			}

			EvalCount := evalCount
			EvalDuration := evalDuration

			avgTokensPerSecond := totalTokensPerSecond / float64(iterations)

			benchmarkResult = &BenchmarkResult{
				ModelName:       modelName,
				Timestamp:       time.Now().Unix(),
				Duration:        time.Since(start).Seconds(),
				EvalCount:       EvalCount,
				EvalDuration:    int64(EvalDuration),
				TokensPerSecond: avgTokensPerSecond,
				Iterations:      iterations,
				SysInfo:         sysinfo,
				GPUInfo:         gpuinfo,
				OllamaVersion:   ollamaVersion,
				ClientType:      "ollamark-gui",
				ClientVersion:   clientVersion,
				IP:              getIPAddress(),
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
			benchmarkButton.SetText("Benchmark")
			benchmarkButton.Enable()
			submitButton.Show()
			submitButton.Enable()
		}()
	}

	submitButton = widget.NewButton("Share Benchmark", nil)
	linkButton = widget.NewButton("View on Ollamark.com", nil)
	linkButton.Hide()

	submitButton.OnTapped = func() {
		if benchmarkResult != nil {
			subEndpoint := os.Getenv("OLLAMARK_API")
			secretKey := os.Getenv("KEY")
			publicKey, err := LoadPublicKey()
			if err != nil {
				resultLabel.SetText("Error loading public key: " + err.Error())
				return
			}

			// Generate AES key
			aesKey, err := generateAESKey()
			if err != nil {
				resultLabel.SetText("Error generating AES key: " + err.Error())
				return
			}

			var submissionID = generateUUID()

			// Generate JWT token
			jwtToken, err := generateJWT(submissionID)
			if err != nil {
				resultLabel.SetText("Error generating JWT token: " + err.Error())
				return
			}

			// Request proof-of-work challenge
			challenge, err := requestProofOfWorkChallenge(subEndpoint)
			if err != nil {
				resultLabel.SetText("Error requesting proof-of-work challenge: " + err.Error())
				return
			}

			// Solve proof-of-work challenge
			powNonce, err := solveProofOfWork(challenge)
			if err != nil {
				resultLabel.SetText("Error solving proof-of-work challenge: " + err.Error())
				return
			}

			// Include proof-of-work solution in the benchmark result
			benchmarkResult.ProofOfWork = ProofOfWorkSolution{
				Challenge:  challenge.Challenge,
				Nonce:      powNonce,
				Timestamp:  challenge.Timestamp,
				Difficulty: challenge.Difficulty,
			}

			// Encrypt benchmark result with AES key
			jsonData, _ := json.Marshal(benchmarkResult)
			nonce, encryptedData, err := encryptAESGCM(aesKey, jsonData)
			if err != nil {
				resultLabel.SetText("Error encrypting data with AES: " + err.Error())
				return
			}

			// Encrypt AES key with RSA public key
			encryptedAESKey, err := encryptRSA(publicKey, aesKey)
			if err != nil {
				resultLabel.SetText("Error encrypting AES key: " + err.Error())
				return
			}

			// Prepare payload
			payload := map[string]interface{}{
				"data":          base64.StdEncoding.EncodeToString(encryptedData),
				"nonce":         base64.StdEncoding.EncodeToString(nonce),
				"encrypted_key": base64.StdEncoding.EncodeToString(encryptedAESKey),
			}

			payloadBytes, _ := json.Marshal(payload)

			// Sign the UUID
			signature := signUUID(submissionID, secretKey)

			// Create and send the request
			req, err := http.NewRequest("POST", subEndpoint+"/api/submit-benchmark", bytes.NewBuffer(payloadBytes))
			if err != nil {
				resultLabel.SetText("Error submitting benchmark! Try again!")
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+jwtToken)
			req.Header.Set("X-Submission-ID", submissionID)
			req.Header.Set("X-Signature", signature)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				resultLabel.SetText("Error submitting benchmark: " + err.Error())
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				resultLabel.SetText("Error submitting benchmark: " + string(body))
				return
			}

			resultLabel.SetText("Benchmark submitted successfully!")
			submitButton.Hide()
			// set linkButton link
			linkButton.OnTapped = func() {
				submissionURL, err := url.Parse(fmt.Sprintf("https://ollamark.com/marks/%s", submissionID))
				if err != nil {
					fmt.Printf("Failed to parse URL: %v\n", err)
					return
				}
				fyne.CurrentApp().OpenURL(submissionURL)
			}
			linkButton.Show()
		}
	}

	submitButton.Hide()
	linkButton.Hide()

	// border/group around systext and gputext
	sysInfoGroup := container.NewVBox(ollamaVersionText, sysText, gpuText)
	sysInfoGroupLabel := widget.NewLabel("System Information")
	sysInfoGroupLabel.TextStyle = fyne.TextStyle{Bold: true}
	sysInfoGroup = container.NewBorder(sysInfoGroupLabel, nil, nil, nil, sysInfoGroup)

	content := container.NewVBox(
		logo,
		sysInfoGroup,
		titleLabel,
		apiEntry,
		title2Label,
		modelSelect,
		iterationsLabel,
		iterationsSlider,
		gif,
		// widget.NewSeparator(),
		tokensPerSecondText,
		tpsText,
		resultLabel,
		progressBar,
		// widget.NewSeparator(),
		benchmarkButton,
		submitButton,
		linkButton,
	)

	// Wrap the content with a padded container
	paddedContent := container.NewPadded(container.NewPadded(content))

	w.SetContent(paddedContent)
	w.ShowAndRun()
}

func contains(models []ModelInfo, modelName string) bool {
	for _, model := range models {
		if model.Name == modelName {
			return true
		}
	}
	return false
}

func runBenchmarkCLI(modelName string, submit bool, ollamaAPI string, iterations int) {
	ollamaAPIURL := ollamaAPI

	var totalTokensPerSecond float64
	var evalCount int
	var evalDuration float64

	// modelName needs to match a model name in MODELS
	if !contains(globalModels, modelName) {
		fmt.Println("Model not supported. Please use a supported model from the list:", globalModels)
		return
	}

	sysinfo, err := getSysInfo()
	if err != nil {
		// fmt.Println("Error:", err)
		return
	}
	fmt.Printf("CPU: %+v\n", sysinfo.CPUName)
	fmt.Printf("Memory: %+v\n", sysinfo.Memory)
	fmt.Printf("OS: %+v\n", sysinfo.OS)
	fmt.Printf("Kernel: %+v\n", sysinfo.Kernel)

	gpuinfo, err := getGPUInfo()
	if err != nil {
		// fmt.Println("Error:", err)
		return
	}
	fmt.Printf("GPU Name: %+v\n", gpuinfo.Name)
	fmt.Printf("Driver Version: %+v\n", gpuinfo.DriverVersion)
	fmt.Printf("GPU Memory: %+v\n", gpuinfo.Memory)

	modelRequest := ModelRequest{
		Name: modelName,
	}
	jsonData, _ := json.Marshal(modelRequest)
	fullURL := ollamaAPI + "/api/pull"
	fmt.Println("Pulling model " + modelName + ", Please wait...")
	resp, err := http.Post(fullURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error pulling model:", string(body))
		return
	}

	fmt.Println("Model pulled successfully")
	fmt.Println("Benchmarking...")
	start := time.Now()

	for i := 0; i < iterations; i++ {
		requestBody := OllamaRequest{
			ModelName: modelName,
			Prompt:    "Tell me about Llamas in 500 words.",
		}

		jsonData, _ := json.Marshal(requestBody)
		resp, err := http.Post(ollamaAPIURL+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		defer resp.Body.Close()

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
		evalCount = response.EvalCount
		evalDuration = float64(response.EvalDuration) / 1e9

	}

	EvalCount := evalCount
	EvalDuration := evalDuration
	avgTokensPerSecond := totalTokensPerSecond / float64(iterations)

	fmt.Printf("\nBenchmark completed for %s\n", modelName)
	fmt.Printf("Average Tokens per second: %.2f\n", avgTokensPerSecond)

	sysinfo, _ = getSysInfo()
	gpuinfo, _ = getGPUInfo()

	benchmarkResult := &BenchmarkResult{
		ModelName:       modelName,
		Timestamp:       time.Now().Unix(),
		Duration:        time.Since(start).Seconds(),
		EvalCount:       EvalCount,
		EvalDuration:    int64(EvalDuration),
		TokensPerSecond: avgTokensPerSecond,
		Iterations:      iterations,
		SysInfo:         sysinfo,
		GPUInfo:         gpuinfo,
		OllamaVersion:   getOllamaVersion(),
		ClientType:      "ollamark-cli",
		ClientVersion:   clientVersion,
		IP:              getIPAddress(),
	}

	if submit {
		submitBenchmark(benchmarkResult)
	} else {
		fmt.Println("Benchmark results not submitted.")
	}
}

func generateJWT(nonce string) (string, error) {
	secretKey := os.Getenv("KEY")
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(time.Minute * 1).Unix(), // Token expires in 1 minute
		"nonce": nonce,
	})

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func submitBenchmark(benchmarkResult *BenchmarkResult) error {
	apiEndpoint := os.Getenv("OLLAMARK_API")
	secretKey := os.Getenv("KEY")
	publicKey, err := LoadPublicKey()
	if err != nil {
		return fmt.Errorf("error loading public key: %v", err)
	}

	// Generate AES key
	aesKey, err := generateAESKey()
	if err != nil {
		return fmt.Errorf("error generating AES key: %v", err)
	}

	var submissionID = generateUUID()

	// Generate JWT token
	jwtToken, err := generateJWT(submissionID)
	if err != nil {
		return fmt.Errorf("error generating JWT token: %v", err)
	}

	// Request proof-of-work challenge
	challenge, err := requestProofOfWorkChallenge(apiEndpoint)
	if err != nil {
		return fmt.Errorf("error requesting proof-of-work challenge: %v", err)
	}

	// Solve proof-of-work challenge
	powNonce, err := solveProofOfWork(challenge)
	if err != nil {
		return fmt.Errorf("error solving proof-of-work challenge: %v", err)
	}

	// Include proof-of-work solution in the benchmark result
	benchmarkResult.ProofOfWork = ProofOfWorkSolution{
		Challenge:  challenge.Challenge,
		Nonce:      powNonce,
		Timestamp:  challenge.Timestamp,
		Difficulty: challenge.Difficulty,
	}

	// Encrypt benchmark result with AES key
	jsonData, _ := json.Marshal(benchmarkResult)
	nonce, encryptedData, err := encryptAESGCM(aesKey, jsonData)
	if err != nil {
		return fmt.Errorf("error encrypting data with AES: %v", err)
	}

	// Encrypt AES key with RSA public key
	encryptedAESKey, err := encryptRSA(publicKey, aesKey)
	if err != nil {
		return fmt.Errorf("error encrypting AES key: %v", err)
	}

	// Prepare payload
	payload := map[string]interface{}{
		"data":          base64.StdEncoding.EncodeToString(encryptedData),
		"nonce":         base64.StdEncoding.EncodeToString(nonce),
		"encrypted_key": base64.StdEncoding.EncodeToString(encryptedAESKey),
	}

	payloadBytes, _ := json.Marshal(payload)

	// Sign the UUID
	signature := signUUID(submissionID, secretKey)

	// Create and send the request
	req, err := http.NewRequest("POST", apiEndpoint+"/api/submit-benchmark", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("error submitting benchmark! %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("X-Submission-ID", submissionID)
	req.Header.Set("X-Signature", signature)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error submitting benchmark: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server responded with status %d: %s", resp.StatusCode, body)
	}

	fmt.Printf("Benchmark submitted successfully! View it at: https://ollamark.com/marks/%s\n", submissionID)
	return nil
}
