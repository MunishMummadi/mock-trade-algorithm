# Mock Trade Algorithm

A Go application for simulating and executing trading algorithms using Alpaca API with mock trading capabilities.

## Overview

This project provides a framework for developing, testing, and running trading algorithms without risking real capital. It connects to the Alpaca trading API and uses a local SQLite database to track mock trades and user information.

## Project Structure

```
.
├── alpaca/         # Alpaca API client code
├── config/         # Configuration management
├── database/       # Database connection and operations
├── models/         # Data models (users, trades)
├── go.mod          # Go module definition
├── go.sum          # Go module checksums
└── main.go         # Application entry point
```

## Dependencies

- [alpaca-trade-api-go](https://github.com/alpacahq/alpaca-trade-api-go) - Go client for Alpaca trading API
- [godotenv](https://github.com/joho/godotenv) - For loading environment variables
- [go-sqlite3](https://github.com/mattn/go-sqlite3) - SQLite driver for Go
- [decimal](https://github.com/shopspring/decimal) - For precise decimal calculations

## Getting Started

### Prerequisites

- Go 1.24 or later
- Alpaca API credentials

### Configuration

1. Copy the example environment file:

```bash
cp config/.env.example config/.env
```

2. Edit `config/.env` with your Alpaca API credentials:

```
ALPACA_API_KEY=your_api_key_here
ALPACA_API_SECRET=your_api_secret_here
```

### Building and Running

```bash
# Build the application
go build -o mock-trade

# Run the application
./mock-trade
```

## Features

- Connect to Alpaca trading API
- Simulate trades without risking real capital
- Store trade history in local SQLite database
- Track user profiles and performance
