# ai-cli

## 🚀 Overview

`ai-cli` is a command-line interface tool designed to turn natural language into proper Linux and Windows commands. It enables users to interact with AI models effortlessly, providing an intuitive interface for generating commands based on user input.

## 🌟 Features

- **Natural Language Processing**: Convert user input in natural language to executable Linux and Windows commands.
- **Cross-Platform Support**: Build binaries for Linux, Windows, and macOS (`darwin`) with `amd64` architecture.
- **AI Integration**: Support for multiple AI APIs including:
  - **Ollama API**: Default integration for fast and efficient AI responses.
  - **OpenAI API**: Optional integration with API key validation.
- **Command History**: Built-in history management feature to keep track of previously generated commands.

## 📦 Installation

### Binary Installation

You can install the binary directly by running the following command:

```bash
curl -sSL https://raw.githubusercontent.com/rohitlohar45/ai-cli/master/install.sh | bash
```

### Windows Installation

For Windows users, download the latest release binary from [GitHub Releases](https://github.com/rohitlohar45/ai-cli/releases), and add the binary to your system’s PATH for easy access.

To add the binary to your PATH:

1. Right-click on 'This PC' and go to 'Properties'.
2. Click on 'Advanced System Settings'.
3. Go to 'Environment Variables' and edit the PATH variable by adding the directory where the `ai-cli` binary is located.

## 🛠 Build from Source

If you prefer to build from source, follow these steps:

1. Clone the repository:

   ```bash
   git clone https://github.com/rohitlohar45/ai-cli.git
   cd ai-cli
   ```

2. Install Go (if not already installed) and set up your environment.

3. Build the binaries:
   ```bash
   make build
   ```

## 📝 Usage

To use `ai-cli`, simply type your command in natural language, and the CLI will translate it into the appropriate Linux or Windows command. Here are a few examples:

- **Example 1**: Convert "list all files in the current directory" into `ls` (Linux) or `dir` (Windows).
- **Example 2**: Convert "create a new directory called projects" into `mkdir projects`.

Run the command:

```bash
./ai-cli "your natural language command here"
```

### Using the Ollama API

To use `ai-cli` with the Ollama API, make sure you have the Ollama container running on the default port (127.0.0.1:11434). You can run the Ollama Docker container using the following command:

```bash
docker run --rm -d -p 11434:11434 ollama/ollama
```

Once the container is running, `ai-cli` will communicate with the Ollama API to generate responses for your commands.

### Using the OpenAI API

To switch to the OpenAI API, you need to provide your OpenAI API key. You can set the API key via the command line or environment variables.

## 📚 Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository.
2. Create your feature branch:
   ```bash
   git checkout -b feature/YourFeature
   ```
3. Commit your changes:
   ```bash
   git commit -m "Add some feature"
   ```
4. Push to the branch:
   ```bash
   git push origin feature/YourFeature
   ```
5. Open a Pull Request.

## ⚠ Known Issues

- No reported issues yet. Please open an issue on GitHub if you encounter any problems.
