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
        Ollama API endpoint (default "http://localhost:11434/api/generate")
  -s    Submit benchmark results to Ollamark (default false)
Examples:
  For Ollamark GUI mode:
      ollamark (no flags)
  For Ollamark CLI mode:
      ollamark -m llama3 -i 10
      ollamark -m phi3
      ollamark -m phi3 -s -o http://localhost:11434/api/generate
```

### Example
```bash
./ollamark -m llama3 -s true -i 5 -o "http://localhost:11434/api/generate"
```

This command will benchmark the model "llama3" for 5 iterations, submit the results, and use the specified API endpoint to interface with Ollama.

## Configuration
The CLI checks for command-line arguments and if provided, Ollamark runs in CLI mode. If no arguments are provided, it defaults to the Ollamark GUI application.


## Additional Information
- Ensure the `.env` file is correctly configured as it loads environment variables crucial for the application.
- The application can also be run as a Fyne GUI application if no CLI flags are provided.

Example .env file:
```
API_ENDPOINT=http://localhost:11434/api/generate
API_KEY=
```
## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

## License
This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments
- [Ollama](https://ollama.com/) for providing the Ollama API.
- [Fyne](https://fyne.io/) for providing the Fyne GUI framework.
