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
- 📝 Import expenses via web interface (CSV, JSON) with automatic or interactive mapping

## Data Privacy

ExpenseTrace is designed with privacy in mind:

- All data is stored locally in a SQLite database
- No external API calls or data sharing
- No bank account connections required
- Full control over your financial data

## Installation

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

3. Create a configuration file (`expensetrace.yml`):

```yaml
db:
  source: expenses.db
  # Optional: Connection pool settings
  # max_open_conns: 25
  # max_idle_conns: 5
  # conn_max_lifetime: 1h
  # conn_max_idle_time: 5m

  # Optional: SQLite PRAGMA settings for better performance
  # journal_mode: WAL
  # synchronous: NORMAL
  # cache_size: -8000
  # busy_timeout: 5000

logger:
  level: info
  format: text
  output: stdout
```

4. Start the web interface:

```bash
expensetrace web
```

### Using Docker Compose (Recommended)

ExpenseTrace can be run using Docker. The simplest way is to use Docker Compose:

1. Create a `docker-compose.yml` file:

```yaml
services:
  expensetrace:
    image: gustavocaso/expensetrace:latest
    environment:
      EXPENSETRACE_DB: /data/expenses.db               # Path to the SQLite database file inside the container
      EXPENSETRACE_PORT: 8081                          # Port the application will listen on inside the container
      EXPENSETRACE_LOG_LEVEL: info                     # Log level: debug, info, warn, error
      EXPENSETRACE_LOG_FORMAT: text                    # Log format: text or json
      EXPENSETRACE_LOG_OUTPUT: stdout                  # Log output: stdout, stderr, or file path
    ports:
      - "8082:8081"                                    # Maps container port 8081 to host port 8082
    volumes:
      - ./:/data                                        # Mounts the current directory to /data in the container.
```

2. Start the service:

```bash
docker compose up
```

The service will be available at `http://localhost:8082`. The configuration file and database will be persisted on the host machine.

## Environment Variables

ExpenseTrace can be configured entirely through environment variables, which override values from the configuration file. This makes it ideal for containerized deployments.

### Configuration

- `EXPENSETRACE_CONFIG`: Path to configuration file (default: `expensetrace.yml`)

### Database Configuration

#### Basic Configuration
- `EXPENSETRACE_DB`: Path to SQLite database file (default: `expensetrace.db`)

#### Connection Pool Settings
- `EXPENSETRACE_DB_MAX_OPEN_CONNS`: Maximum number of open connections to the database (default: unlimited)
- `EXPENSETRACE_DB_MAX_IDLE_CONNS`: Maximum number of idle connections (default: 2)
- `EXPENSETRACE_DB_CONN_MAX_LIFETIME`: Maximum lifetime of a connection (e.g., `1h`, `30m`, `5s`)
- `EXPENSETRACE_DB_CONN_MAX_IDLE_TIME`: Maximum idle time for a connection (e.g., `5m`, `10m`)

#### SQLite PRAGMA Settings
- `EXPENSETRACE_DB_JOURNAL_MODE`: SQLite journal mode - `DELETE`, `TRUNCATE`, `PERSIST`, `MEMORY`, `WAL`, `OFF` (default: `DELETE`)
  - `WAL` (Write-Ahead Logging) is recommended for better concurrent access
- `EXPENSETRACE_DB_SYNCHRONOUS`: SQLite synchronous mode - `OFF`, `NORMAL`, `FULL`, `EXTRA` (default: `FULL`)
  - `NORMAL` provides better performance while maintaining good safety
- `EXPENSETRACE_DB_CACHE_SIZE`: SQLite cache size in kilobytes (negative values) or pages (positive values) (e.g., `-2000` for 2MB)
- `EXPENSETRACE_DB_BUSY_TIMEOUT`: Timeout in milliseconds when database is locked (e.g., `5000` for 5 seconds)
- `EXPENSETRACE_DB_WAL_AUTOCHECKPOINT`: WAL auto-checkpoint interval in pages (default: 1000)
- `EXPENSETRACE_DB_TEMP_STORE`: Temporary storage location - `DEFAULT`, `FILE`, `MEMORY`

### Web Server Configuration

- `EXPENSETRACE_PORT`: Web server port (default: `8080`)
- `EXPENSETRACE_TIMEOUT`: Server timeout duration (default: `5s`)
- `EXPENSETRACE_ALLOW_EMBEDDING`: Allow iframe embedding - set to `true` to enable (default: `false`)

### Logging Configuration

- `EXPENSETRACE_LOG_LEVEL`: Log level - `debug`, `info`, `warn`, `error` (default: `info`)
- `EXPENSETRACE_LOG_FORMAT`: Log format - `text` or `json` (default: `text`)
- `EXPENSETRACE_LOG_OUTPUT`: Log output - `stdout`, `stderr`, or file path (default: `stdout`)

### Recommended Production Settings

For production deployments with better performance and concurrency, consider these settings:

```bash
EXPENSETRACE_DB_JOURNAL_MODE=WAL
EXPENSETRACE_DB_SYNCHRONOUS=NORMAL
EXPENSETRACE_DB_CACHE_SIZE=-8000
EXPENSETRACE_DB_BUSY_TIMEOUT=5000
EXPENSETRACE_DB_MAX_OPEN_CONNS=25
EXPENSETRACE_DB_MAX_IDLE_CONNS=5
EXPENSETRACE_DB_CONN_MAX_LIFETIME=1h
```


### Importing Expenses

ExpenseTrace provides a flexible import system through the web interface that supports both automatic and interactive workflows:

#### Automatic Import

##### CSV

For supported banking providers, ExpenseTrace automatically handles field mapping. Simply name your CSV file with the provider prefix:

- **EVO**: `evo_transactions.csv`
- **Revolut**: `revolut_transactions.csv`
- **Bankinter**: `bankinter_transactions.csv`

The system will automatically detect the provider and parse the CSV correctly.

#### JSON
Uploading a json file that containts an array with objects containing `source`, `date`, `description`, `amount`, and `currency` fields

Example JSON format:
```json
[
  {
    "source": "MyBank",
    "date": "2024-01-15T10:30:00Z",
    "description": "grocery store",
    "amount": -5000,
    "currency": "EUR"
  }
]
```

#### Interactive Import

For custom CSV or JSON files, ExpenseTrace provides an interactive 3-step import process:

1. **Upload & Preview**: Upload your file and see a preview of the data
2. **Field Mapping**: Map your file's columns to expense fields:
   - Date column
   - Description column
   - Amount column
   - Currency column
   - Source (custom name for your data source)
3. **Review & Confirm**: Preview the parsed expenses and confirm the import

#### Category Pattern Matching

ExpenseTrace uses regular expressions (regex) to automatically categorize your expenses based on transaction descriptions. Here's how to effectively use pattern matching:

```yaml
  - "Groceries" -> "supermarket|grocery|food market"  # Matches any of these terms
  - "Transportation" -> "uber|taxi|metro|bus|train"        # Matches various transport services
  - "Entertainment" -> "netflix|spotify|cinema|theater"   # Matches entertainment services
  - "Utilities" -> "electricity|water|gas|internet"   # Matches utility bills
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
  - "Online Shopping" -> "amazon.*|ebay.*|walmart.*"  # Matches any transaction from these retailers
  - "Dining Out" -> "^restaurant|^cafe|^bar|^pizza"  # Matches only if these terms appear at the start
  - "Subscriptions" -> ".*subscription$|.*membership$"  # Matches if these terms appear at the end
  - "Healthcare" -> "pharmacy|doctor|hospital|medical"  # Matches healthcare-related expenses
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Developing locally

When working on the web interface you can use `make run_web` to run the web server. This set the ENV variable `EXPENSE_LIVERELOAD=true`. That ensures changes to files in the `router/templates` folder are pickup when reloading the browser.

Making changes to the HTTP hanlder code, requires restarting the server.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
