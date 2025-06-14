# Go-BitTorrent

A simple BitTorrent implementation featuring both a cross-platform GUI client (written in Go) and a Node.js tracker server. This project is intended for educational and experimental use and is the client is probably full of bugs.

## Overview

This repository contains two main components:

- **Client** (`client/`):
  - Written in Go, with a modern GUI using Fyne.
  - Supports downloading and seeding torrents.
  - Features include:
    - Peer-to-peer file transfer
    - Custom AES traffic encryption toggle
    - Real-time progress and peer status
    - Easy torrent file selection and management
    - HTTP and UDP tracker announce logic

- **Tracker** (`bittorrent-tracker/`):
  - Node.js BitTorrent tracker, originally based on [webtorrent/bittorrent-tracker](https://github.com/webtorrent/bittorrent-tracker).
  - Extended with an SQLite database for persistent peer and torrent state.
  - Supports HTTP and UDP tracker protocols.

## Features

### Client
- Download and seed torrents with a simple GUI
- Toggle AES encryption for peer traffic
- View leeching and seeding status
- Cross-platform (Windows, Linux, macOS)

### Tracker
- Tracks peers and torrents using HTTP, UDP, and WebSocket
- Persists peer/torrent info in an SQLite database (custom addition)
- Lightweight and easy to run

## Getting Started

### Client (Go)
1. Install Go 1.18+ and [Fyne](https://fyne.io/)
2. Build and run:
   ```sh
   cd client
   go build -o client.exe
   ./client.exe
   ```

### Tracker (Node.js)
1. Install Node.js 16+
2. Install dependencies:
   ```sh
   cd bittorrent-tracker
   npm install
   ```
3. Start the tracker:
   ```sh
   node src/server.js
   ```

## Notes
- The tracker code is largely based on the open-source [webtorrent/bittorrent-tracker](https://github.com/webtorrent/bittorrent-tracker) project, with the main addition being the use of an SQLite database for persistent storage.
- The client is an original Go implementation with a modern GUI.
