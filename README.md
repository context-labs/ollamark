# Ollamark README

## Overview
Ollamark and Ollamark CLI is a command-line/UI interface for benchmarking models using the Ollama API. It allows users to specify the model to benchmark, whether to submit the benchmark results, and the API endpoint.

**This is a WIP and is subject to change! (Submitting benchmarks is not completed yet.)**

## Installation
Ensure you have Go installed on your system. Clone the repository and build the project using:
```bash
go build
```

## Usage
Run the Ollamark CLI using the following flags to customize the benchmarking process:

### Flags
- `-m`: Model name to benchmark. Default is `"llama3"`.
- `-s`: Submit benchmark results. It accepts a boolean value. Default is `false`.
- `-o`: Ollama API endpoint. Default is `"http://localhost:11434/api/generate"`.

### Example
```bash
./ollamark -m llama3 -s true -o "http://localhost:11434/api/generate"
```

This command will benchmark the model "llama3", submit the results, and use the specified API endpoint.

## Configuration
The CLI checks for command-line arguments and if provided, it runs in CLI mode. If no arguments are provided, it defaults to the Ollamark GUI application.

Refer to the code snippet for parsing CLI arguments:

```55:59:main.go
	// Parse command-line arguments (Ollamark CLI)
	modelPtr := flag.String("m", "llama3", "Model name to benchmark")
	submitPtr := flag.Bool("s", false, "Submit benchmark results")
	ollamaPtr := flag.String("o", "http://localhost:11434/api/generate", "Ollama API endpoint")
	flag.Parse()
```


## Additional Information
- Ensure the `.env` file is correctly configured as it loads environment variables crucial for the application.
- The application can also be run as a Fyne GUI application if no CLI flags are provided.


Example .env file:
```
API_ENDPOINT=http://localhost:11434/api/generate
API_KEY=
```

For more details on the implementation, refer to the main function in the Go source file:

```48:276:main.go
func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
```
