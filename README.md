# Download Manager in Golang

## Overview

This project is a **Download Manager** implemented in **Golang** with a **Text-Based User Interface (TUI)**. It supports multiple queues, concurrent downloads, speed limitations, scheduled downloads, and essential download control functionalities.

## Features

### 1. Download Queues

- Each queue contains multiple downloads.
- Configurable settings per queue:
  - **Storage folder** (e.g., `~/Downloads`).
  - **Max simultaneous downloads** (e.g., 3 at a time).
  - **Max bandwidth limit** (e.g., 500 KB/s or unlimited).
  - **Active time range** (e.g., downloads allowed only from `22:00-06:00`).
  - **Max retry attempts** (e.g., set to `0` for no retries).

### 2. Download Management

- Add new downloads via the first tab (**New Download Form**).
- Download statuses:
  - Downloading (shows **progress** and **speed**).
  - Paused.
  - Completed.
  - Failed.
- Support for **Pause, Resume, Cancel, and Retry**.

### 3. Text-Based User Interface (TUI)

- Three main tabs:
  1. **New Download** (Add new files for download).
  2. **Downloads List** (View all downloads and their statuses).
  3. **Queues List** (Manage queues, settings, and limits).
- **Keyboard Shortcuts** for easy navigation and control.
- **Persistent state** (downloads and settings are saved across sessions).

## Technologies & Concepts

- **Concurrency in Golang** (Goroutines & Channels for parallel downloads).
- **TUI Implementation** (Using `tview` or `bubble tea`).
- **Networking & File Handling** (HTTP requests and multi-part downloads using `Accept-Ranges`).
- **Golang Structs & Methods** (For managing downloads and queues efficiently).
- **Error Handling & Retries** (Automatic and manual retry options).
- **Configuration Management** (Load & save settings for persistence).

## Installation & Usage

### Prerequisites

- Install **Go (>=1.18)**
- Clone the repository:
  ```sh
  git clone https://github.com/nimaazarr/DM.git
  cd DM
  ```
- Install dependencies:
  ```sh
  go mod tidy
  ```

### Running the Application

To start the **Download Manager**, run:

```sh
go run main.go
```

### Keyboard Shortcuts

- **F1** → Add New Download
- **F2** → View Downloads List
- **F3** → Manage Queues
- **Arrow Keys** → Navigate lists
- **P** → Pause/Resume download
- **D** → Delete download
- **R** → Retry failed download

## Project Structure

```
download-manager/
│── cmd/                # CLI application entry point
│   ├── main.go         # Main execution file
│
│── internal/           # Core business logic (not exposed outside)
│   ├── download/       # Download management logic
│   │   ├── manager.go  # Handles multiple downloads
│   │   ├── worker.go   # Concurrent worker logic
│   │   ├── queue.go    # Queue management (pause/resume/download scheduling)
│   │   ├── file.go     # File operations
│   │   ├── speed.go    # Speed control & rate limiting
│   │   ├── schedule.go # Handles scheduled downloads
│   │
│   ├── tui/            # Text-based UI logic
│   │   ├── app.go      # Main TUI application logic
│   │   ├── layout.go   # TUI layout and widgets
│   │   ├── event.go    # Handles key bindings and events
│   │
│   ├── config/         # Configuration and settings
│   │   ├── config.go   # Reads and manages application settings
│   │
│   ├── utils/          # Utility functions
│   │   ├── logger.go   # Logging utilities
│   │   ├── helpers.go  # Miscellaneous helpers
│
│── api/                # Optional: API server for remote control
│   ├── server.go       # HTTP server for controlling downloads remotely
│   ├── routes.go       # API route handlers
│
│── pkg/                # Reusable packages (if needed)
│   ├── network/        # Networking utilities
│   ├── storage/        # File storage and handling
│
│── scripts/            # Scripts for automation
│   ├── build.sh        # Script to build the project
│   ├── test.sh         # Script to run tests
│
│── test/               # Test files
│   ├── download_test.go  # Tests for the download logic
│   ├── tui_test.go       # Tests for the TUI logic
│
│── .env                # Environment variables (if needed)
│── .gitignore          # Git ignore file
│── go.mod              # Go module file
│── go.sum              # Go dependencies checksum
│── README.md           # Documentation
```

## License

This project is licensed under the **MIT License**.

## Author

Nima Azar
Kasra Siavashpour
Ardalan Siavashpour

