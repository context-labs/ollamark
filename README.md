# Ollamark

### By Carsen Klock (https://twitter.com/carsenklock) for Ollama (https://ollama.com/) Benchmarking!

## Overview
Ollamark and Ollamark CLI is a command-line/UI interface for benchmarking models using the Ollama API. It allows users to specify the model to benchmark, whether to submit and share the benchmark results, the API endpoint, and number of iterations.

**This is a WIP and is subject to change! Ollamark.com coming soon!**

## Building
Ensure you have Go installed on your system. Clone the repository and build the project using:
```bash
go build
```

## Installing
- Download and Install Ollama from https://ollama.com/download
- Ollama will start automatically in the background
- Run Ollamark with flags to start the benchmarking process in CLI mode or without flags to run in GUI mode

## Usage
Run the Ollamark CLI using the following flags to customize the benchmarking process:

### Flags
- `-m`: Model name to benchmark. Default is `"llama3"`.
- `-s`: Submit benchmark results. It accepts a boolean value. Default is `false`.
- `-o`: Ollama API endpoint. Default is `"http://localhost:11434"`.
- `-i`: Number of iterations to run the benchmark. Default is `2`.
- `-h` or `-help`: Display the help message below.

```
Usage: ollamark [options]
Options:
  -i int
        Number of benchmark iterations (default 2)
  -m string
        Model name to benchmark (default "llama3")
  -o string
        Ollama API endpoint (default "http://localhost:11434")
  -s    Submit benchmark results to Ollamark (default false)
Examples:
  For Ollamark GUI mode:
      ollamark (no flags)
  For Ollamark CLI mode:
      ollamark -m llama3 -i 10
      ollamark -m phi3
      ollamark -m phi3 -s -o http://localhost:11434
```

### Example
```bash
./ollamark -m llama3 -s -i 5 -o "http://localhost:11434"
```

This command will benchmark the model "llama3" for 5 iterations, submit the results, and use the specified API endpoint to interface with Ollama.

## Configuration
The CLI checks for command-line arguments and if provided, Ollamark runs in CLI mode. If no arguments are provided, it defaults to the Ollamark GUI application.

## Additional Information for Building/Forking
- Ensure the `.env` file is correctly configured as it loads environment variables crucial for the application.
- The application can also be run as a Fyne GUI application if no CLI flags are provided.

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## License
This project is licensed under the MIT License - see the LICENSE file for details.

## Author
- Carsen Klock (https://twitter.com/carsenklock)

## Acknowledgments
- [Ollama](https://ollama.com/) for providing Ollama.
- [GoLang](https://golang.org/) for providing the Go programming language.
- [Fyne](https://fyne.io/) for providing the Fyne GUI framework.
- [Gin](https://gin.github.io/) for providing the Gin HTTP web framework.
