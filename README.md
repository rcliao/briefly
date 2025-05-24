# Briefly: AI-Powered Digest Generator

Briefly is a command-line application written in Go that takes a Markdown file containing a list of URLs, fetches the content from each URL, summarizes the text using a Large Language Model (LLM) via the Gemini API, and then generates a cohesive Markdown-formatted digest of all the summarized content.

## Features

- Reads URLs from a specified Markdown file.
- Fetches and parses HTML content from each URL to extract the main text.
- Saves extracted text to a local temporary folder.
- Summarizes each article's text using the Gemini API.
- Generates a final, cohesive Markdown digest from all individual summaries, also using the Gemini API.
- Includes citations (links back to original URLs) in the final digest.
- Outputs the final digest to the console and saves it to a Markdown file.
- Configurable via command-line flags for API key, model, input/output paths.
- Provides detailed logging of its operations.

## Prerequisites

- Go (version 1.18 or higher recommended)
- A Gemini API Key

## Setup

1. **Clone the Repository:**

   ```bash
   git clone https://your-repository-url.git # Replace with your actual repository URL
   cd briefly
   ```

2. **Install Dependencies:**

   ```bash
   go mod tidy
   ```

3. **Set up Gemini API Key:**

   You need to provide your Gemini API key to the application. You can do this in one of two ways:

   - **Environment Variable (Recommended):** Create a file named `.env` in the root of the project directory and add your API key:

     ```
     GEMINI_API_KEY="YOUR_API_KEY_HERE"
     ```

     Make sure `.env` is listed in your `.gitignore` file to prevent committing your key.

   - **Command-Line Flag:** Use the `-api-key` flag when running the application (see Usage section).

## Usage

The application is run from the command line.

```bash
go run main.go -input <path_to_markdown_file> [flags]
```

**Required Flag:**

- `-input <filepath>`: Path to the Markdown file containing the list of URLs to process.

**Optional Flags:**

- `-api-key <key>`: Your Gemini API Key. If not provided, the application will try to read it from the `GEMINI_API_KEY` environment variable (loaded from an `.env` file if present).
- `-model <model_name>`: The Gemini model to use for summarization and digest generation.
  - Default: `gemini-1.5-flash-latest`
- `-temp-path <path>`: Path to the directory where temporary files (fetched article text) will be stored.
  - Default: `./temp_content/`
- `-output-path <path>`: Path to the directory where the final Markdown digest file will be saved. The digest file will be named `digest_YYYY-MM-DD.md`.
  - Default: `./digests/`

**Example:**

```bash
# Using an .env file for the API key
go run main.go -input ./input/my_links.md

# Providing API key via flag and specifying a different model
go run main.go -input ./input/my_links.md -api-key "YOUR_GEMINI_KEY" -model "gemini-1.0-pro"

# Specifying custom temporary and output paths
go run main.go -input ./input/weekly_articles.md -temp-path /tmp/briefly_cache -output-path ./generated_digests
```

## Input File Format

The input file specified with `-input` should be a plain Markdown file. The application will extract all URLs (starting with `http://` or `https://`) found anywhere in this file.

Example `input/links.md`:

```markdown
# Weekly Links

Here are some interesting articles I found this week:

- https://blog.example.com/article-one
- Check this out: https://news.example.org/important-update

Another one: (https://another.example.net/research-paper)
```

## Workflow

1. **Configuration & Initialization**: The application starts, loads configuration from flags and environment variables, and sets up logging.
2. **Read Input**: Reads the specified Markdown file.
3. **Extract URLs**: Parses the Markdown content to find all URLs.
4. **Validate URLs**: Filters out invalid or non-HTTP/HTTPS URLs.
5. **Process Each URL**: For each valid URL:
   - **Fetch Content**: Retrieves the HTML content from the URL.
   - **Extract Text**: Parses the HTML to extract the main textual content, removing boilerplate like navigation, ads, etc.
   - **Save Text**: Saves the extracted plain text to a file in the configured temporary directory.
   - **Summarize Text**: Sends the extracted text to the Gemini API to generate a concise summary.
6. **Generate Final Digest**:
   - Collects all individual summaries (and their source URLs).
   - Sends the collected summaries to the Gemini API with a prompt to create a single, cohesive, friendly Markdown-formatted weekly digest.
7. **Output Digest**:
   - Prints the final digest to the console.
   - Saves the final digest to a Markdown file (e.g., `digest_YYYY-MM-DD.md`) in the configured output directory.

## Logging

The application logs its progress and any errors to the console. Log messages are prefixed with tags like `[CONFIG]`, `[EXTRACT]`, `[PROCESS]`, `[ERROR]`, etc., to indicate the stage or type of message. Timestamps with microsecond precision are included.

## Project Structure

```
briefly/
├── .env                   # For API key (add to .gitignore)
├── EXECUTION_PLAN.md      # Development plan
├── go.mod                 # Go module file
├── go.sum                 # Go module checksums
├── main.go                # Main application logic
├── README.md              # This file
├── digests/               # Default output directory for generated digests
│   └── digest_YYYY-MM-DD.md
├── input/                 # Example directory for input markdown files
│   └── example.md
├── llmclient/             # Package for interacting with the LLM
│   └── gemini_client.go
└── temp_content/          # Default directory for temporary fetched content
    └── some_sanitized_url.txt
```

## Further Development (Planned from EXECUTION_PLAN.md)

- **Phase 5: Testing and Documentation**
  - Write unit tests for key functions.
  - Write integration tests for the end-to-end workflow.
  - Continuously update documentation.
