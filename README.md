# Startable

`Startable` is a lightweight Go library for managing the lifecycle of asynchronous tasks or services. It provides a simple interface to start, stop, and handle callbacks for pre- and post-start events. Ideal for managing goroutines with proper context cancellation and result handling.

## Features

- Start and stop tasks safely
- Pre-start and post-start callbacks
- Context-aware cancellation
- Simple API for goroutine lifecycle management
- Thread-safe (concurrent calls to Start/Stop)

## Installation

```bash
go get github.com/Amirhos-esm/startable
