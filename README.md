# SIS Project - Checkpoint 2: Concurrent Text Analyzer

## Overview
This application extends the Checkpoint 1 in-memory Text Analyzer with concurrent task processing using a fixed-size **Worker Pool** and a **bounded channel-based queue**. It analyzes text inputs to calculate word, character, rune, and line counts[cite: 75].

## Concurrency Architecture

```text
                 Bounded Queue (chan string, cap: 10)
                 +----------------------------------+
SubmitTask ----> |  [TASK-0001] [TASK-0002] ...     |
                 +----------------------------------+
                                  |
                 +----------------+----------------+
                 |                |                |
                 v                v                v
            Worker 1         Worker 2         Worker 3
                 |                |                |
                 +----------------+----------------+
                                  |
                                  v
                 InMemoryRepository (sync.RWMutex)