# Building a Database in Go
# Building a Database in Go

A simple persistent key-value database built from scratch in Go.

This project explores core database concepts such as file-based storage, writing and reading records, persistence across program restarts, and basic error handling.

## Features

- Store values using key-value pairs
- Persist data to a local database file
- Read saved values after reopening the database
- Update existing keys with newer values
- Handle common file and database errors

## Getting Started

Clone the repository and run:

```bash
go run .
```

Example output:

```text
Written values:
name = Dheer
language = Go

After reopening:
OK: name = Dheer
OK: language = Go
OK: project = build-your-own-db

All persistence tests passed.
```

## Built With

- Go

## Learning Goal

This is a learning project created to better understand how databases manage data on disk and recover it between program runs.