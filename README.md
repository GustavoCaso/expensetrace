# ExpenseTrace

<div align="center">
  <img src="https://raw.githubusercontent.com/GustavoCaso/expensetrace/refs/heads/main/images/logo.png" alt="ExpenseTrace Logo" width="400">
</div>

ExpenseTrace is a privacy-focused expense tracking tool that helps you manage your finances without sharing your banking details with third-party services. Built in Go, it provides a simple yet powerful way to track your expenses, categorize them, and generate insightful reports.

[![Go Report Card](https://goreportcard.com/badge/github.com/GustavoCaso/expensetrace)](https://goreportcard.com/report/github.com/GustavoCaso/expensetrace)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/GustavoCaso/expensetrace)](https://golang.org/)


## Why ExpenseTrace?

In an era where financial data privacy is increasingly important, ExpenseTrace offers a secure alternative to traditional expense tracking apps. Instead of connecting to your bank accounts or sharing sensitive financial information, ExpenseTrace allows you to:

- Import your expenses from CSV or JSON files through the web interface
- Automatically categorize transactions based on customizable regex patterns
- Generate detailed reports and insights
- Access your data through either a terminal interface (TUI) or web interface
- Keep all your financial data local and private

## Features

- 📊 Interactive TUI (Terminal User Interface) for easy navigation
- 🌐 Web interface for visual data exploration with import functionality
- 📈 Detailed financial reports and insights
- 🔒 Local data storage with SQLite
- 🎨 Beautiful terminal output with color coding
- 📝 Import expenses via web interface (CSV, JSON)

## Data Privacy

ExpenseTrace is designed with privacy in mind:

- All data is stored locally in a SQLite database
- No external API calls or data sharing
- No bank account connections required
- Full control over your financial data

## Installation

### Prerequisites

- Go 1.22 or later
- SQLite

### Building from Source

1. Clone the repository:

```bash
git clone https://github.com/GustavoCaso/expensetrace.git
cd expensetrace
```

2. Build the project:

```bash
CGO_ENABLED=1 go build
```

3. Create a configuration file (`expense.yaml`):

```yaml
db: expenses.db
logger:
  level: info
  format: text
  output: stdout
categories:
  - name: "Groceries"
    pattern: "supermarket"
  - name: "Transportation"
    pattern: "uber|taxi|metro"
```

## Environment Variables

ExpenseTrace can be configured entirely through environment variables, which override values from the configuration file. This makes it ideal for containerized deployments.

### Core Configuration

- `EXPENSETRACE_CONFIG`: Path to configuration file (default: `expense.yml`)
- `EXPENSETRACE_DB`: Path to SQLite database file (default: from config file)

### Web Server Configuration

- `EXPENSETRACE_PORT`: Web server port (default: `8080`)
- `EXPENSETRACE_TIMEOUT`: Server timeout duration (default: `5s`)
- `EXPENSETRACE_ALLOW_EMBEDDING`: Allow iframe embedding - set to `true` to enable (default: `false`)

### Logging Configuration

- `EXPENSETRACE_LOG_LEVEL`: Log level - `debug`, `info`, `warn`, `error` (default: `info`)
- `EXPENSETRACE_LOG_FORMAT`: Log format - `text` or `json` (default: `text`)
- `EXPENSETRACE_LOG_OUTPUT`: Log output - `stdout`, `stderr`, or file path (default: `stdout`)

## Usage

ExpenseTrace provides two main interfaces to help you manage your expenses:

### Subcommands

#### `tui`

Launches the interactive terminal user interface:

```bash
expensetrace tui
```

The TUI provides a comprehensive interface for:
- Browsing expense reports by month and year
- Viewing detailed expense breakdowns
- Split view showing reports and detailed expense information
- Navigate using Tab to switch between views and arrow keys to navigate

#### `web`

Launch the web interface:

```bash
expensetrace web
```

### Getting Started

1. **Start with the web interface** for easy setup and data import:
   ```bash
   expensetrace web
   ```
   Then visit `http://localhost:8080` to import your data and configure categories.

2. **Use the TUI** for quick access and terminal-based workflows:
   ```bash
   expensetrace tui
   ```

Both interfaces work with the same underlying data, so you can switch between them as needed.

### Using Docker Compose

ExpenseTrace can be run using Docker Compose for a containerized environment:

1. Create a `docker-compose.yml` file:

```yaml
services:
  expensetrace:
    image: gustavocaso/expensetrace:latest
    environment:
      EXPENSETRACE_CONFIG: /app/data/expensetrace.yml  # Path to the configuration file inside the container
      EXPENSETRACE_DB: /app/data/expenses.db            # Path to the SQLite database file inside the container
      EXPENSETRACE_PORT: 8081                          # Port the application will listen on inside the container
      EXPENSETRACE_LOG_LEVEL: info                     # Log level: debug, info, warn, error
      EXPENSETRACE_LOG_FORMAT: text                    # Log format: text or json
      EXPENSETRACE_LOG_OUTPUT: stdout                  # Log output: stdout, stderr, or file path
      SUBCOMMAND: web
    ports:
      - "8082:8081"                                    # Maps container port 8081 to host port 8082
    volumes:
      - ./:/app/data                                    # Mounts the current directory to /app/data in the container.
```

2. Start the service:

```bash
docker compose up
```

The service will be available at `http://localhost:8082`. The configuration file and database will be persisted on the host machine.

### Configuration

The `expense.yaml` file allows you to:

- Set the database location
- Define expense categories and their matching patterns
- Configure logging settings

#### Category Pattern Matching

ExpenseTrace uses regular expressions (regex) to automatically categorize your expenses based on transaction descriptions. Here's how to effectively use pattern matching:

```yaml
categories:
  - name: "Groceries"
    pattern: "supermarket|grocery|food market"  # Matches any of these terms
  - name: "Transportation"
    pattern: "uber|taxi|metro|bus|train"        # Matches various transport services
  - name: "Entertainment"
    pattern: "netflix|spotify|cinema|theater"   # Matches entertainment services
  - name: "Utilities"
    pattern: "electricity|water|gas|internet"   # Matches utility bills
```

Pattern matching tips:

- Use `|` (pipe) to match multiple patterns: `"pattern1|pattern2"`
- Use `.*` for wildcard matching: `"amazon.*"` matches any `amazon` transaction
- Use `^` for start of string: `"^starbucks"` matches only if `starbucks` is at the start
- Use `$` for end of string: `"subscription$"` matches only if `subscription` is at the end
- Use `\d` for digits: `"payment-\d+"` matches `payement-` followed by any number

Note: Transaction descriptions are automatically converted to lowercase before matching against patterns. Therefore, patterns should be written in lowercase to match correctly.

Example with complex patterns:

```yaml
categories:
  - name: "Online Shopping"
    pattern: "amazon.*|ebay.*|walmart.*"  # Matches any transaction from these retailers
  - name: "Dining Out"
    pattern: "^restaurant|^cafe|^bar|^pizza"  # Matches only if these terms appear at the start
  - name: "Subscriptions"
    pattern: ".*subscription$|.*membership$"  # Matches if these terms appear at the end
  - name: "Healthcare"
    pattern: "pharmacy|doctor|hospital|medical"  # Matches healthcare-related expenses
```

#### Logging Configuration

ExpenseTrace includes comprehensive logging capabilities to help with debugging and monitoring. You can configure logging in your `expense.yaml` file:

```yaml
db: expenses.db
logger:
  level: info      # Log level: debug, info, warn, error
  format: text     # Log format: text or json
  output: stdout   # Output: stdout, stderr, or file path
categories:
  # ... your categories
```

**Logging Options:**

- **Level**: Controls which log messages are shown
  - `debug`: All messages (most verbose)
  - `info`: Informational messages and above (default)
  - `warn`: Warnings and errors only
  - `error`: Error messages only

- **Format**: Controls log message formatting
  - `text`: Human-readable format (default)
  - `json`: Structured JSON format for log aggregation

- **Output**: Where log messages are sent
  - `stdout`: Standard output (default)
  - `stderr`: Standard error
  - File path: e.g., `/var/log/expensetrace.log` for file logging

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
