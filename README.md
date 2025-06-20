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

- Manually import your expenses from CSV or JSON files
- Categorize transactions based on your own rules
- Generate detailed reports and insights
- Access your data through a CLI or web interface
- Keep all your financial data local and private

## Features

- 📊 Interactive TUI (Terminal User Interface) for easy navigation
- 🌐 Web interface for visual data exploration
- 📝 CSV, JSON import functionality
- 📈 Detailed financial reports
- 🔒 Local data storage with SQLite
- 🎨 Beautiful terminal output with color coding

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
categories:
  - name: "Groceries"
    pattern: "supermarket"
  - name: "Transportation"
    pattern: "uber|taxi|metro"
```

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
      SUBCOMMAND: web
    ports:
      - "8082:8081"                                    # Maps container port 8081 to host port 8082
    volumes:
      - ./:/app/data                                    # Mounts the current directory to /app/data in the container.
```

The environment variables control the following aspects of the application:

- `EXPENSETRACE_CONFIG`: Specifies the location of the configuration file inside the container. This file should contain your expense categories and other settings.
- `EXPENSETRACE_DB`: Defines where the SQLite database file will be stored inside the container. This file will persist your expense data.
- `EXPENSETRACE_PORT`: Sets the port number that the application will use to serve the web iterface. Default value `8080`
- `SUBCOMMAND`: Specifies the subcommand that would be use in the container. Default value `web`

2. Start the service:

```bash
docker compose up
```

The service will be available at `http://localhost:8082`. The configuration file and database will be persisted on the host machine.

### Configuration

The `expense.yaml` file allows you to:

- Set the database location
- Define expense categories and their matching patterns

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

## Usage

ExpenseTrace provides several commands to help you manage your expenses:

### Subcommands

#### `tui`

Launches the interactive terminal user interface:

```bash
expensetrace tui
```

The TUI provides a split view showing categories on the left and expenses on the right. Use Tab to switch between views and arrow keys to navigate.

#### `import`

Import expenses from a file:

```bash
expensetrace import -f expenses.csv
```

#### `report`

Generate expense reports:

```bash
expensetrace report [-month MONTH] [-year YEAR] [-v]
```

Options:

- `-month`: Specify month (1-12)
- `-year`: Specify year
- `-v`: Verbose output with detailed expense breakdown

#### `search`

Search for specific expenses:

```bash
expensetrace search -k "keyword" [-v]
```

Options:

- `-k`: Search keyword
- `-v`: Verbose output

#### `category`

Manage expense categories:

```bash
expensetrace category -a inspect|recategorize|migrate [-o output.txt]
```

Options:

- `-a`: Action to perform (inspect, recategorize, or migrate)
- `-o`: Output file for inspection results

#### `web`

Launch the web interface:

```bash
expensetrace web [-p PORT]
```

Options:

- `-p`: Port number (default: 8080)

#### `delete`

Reset the database:

```bash
expensetrace delete
```

⚠️ Warning: This will delete all your expense data!

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
