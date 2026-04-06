> **Notice:** i was too lazy to write this file myself, so i let ai do it, but it probably wrote absolute bullshit

# Architecture Design

Belphegor is a distributed, peer-to-peer clipboard synchronization tool designed for high throughput and low latency. It operates as a mesh network where every node is equal

## 1. High-Level Design

The application follows an **Event-Driven Architecture**. It bridges the operating system's clipboard APIs with a secure network transport layer

### Core Lifecycle
1.  **Detection:** A platform-specific watcher detects a clipboard change.
2.  **Deduplication:** The payload is hashed (xxHash); if the hash matches the last write, the event is discarded to prevent feedback loops
3.  **Broadcasting:** Valid payloads are serialized (Protobuf), encrypted, and broadcast to all connected peers via QUIC streams
4.  **Injection:** Remote peers receive the stream, validate the payload, write data to disk (if it's a file) or memory, and inject it into their local clipboard

## 2. Component Layering

### 2.1. Hardware Abstraction Layer (HAL)
Located in `pkg/clipboard`, this layer isolates OS-specific implementation details. It exposes a unified `Eventful` interface:

*   **Linux (Wayland):**
    *   *Primary:* Native Wayland protocol implementation (`ext-data-control-v1`) using a pure Go client
    *   *Fallback:* shelling out to `wl-clipboard` binaries
*   **Linux (X11):** Native X11 client using `xgb` and `xfixes` extension
*   **Windows:** Direct Win32 API calls via `syscall` (User32/Shell32)
*   **macOS:** Cocoa/AppKit interaction via `purego` (FFI without Cgo)

### 2.2. Core Logic
Located in `internal/node`
*   **Orchestrator:** Manages the lifecycle of the application, binding the clipboard HAL to the network transport
*   **Sync Engine:** Handles the mapping between internal domain events and external clipboard formats (Text, Image, Files)
*   **Storage Strategy:** Large payloads (files/images) are streamed directly to a temporary file store to minimize resident memory usage. Only file paths are passed to the OS clipboard when handling file transfers

### 2.3. Network Layer
Located in `internal/transport` and `internal/discovering`
*   **Transport:** **QUIC** (via `quic-go`). Chosen for its multiplexing capabilities (avoiding Head-of-Line blocking) and 0-RTT/1-RTT handshakes
*   **Security:** Enforced **TLS 1.3**. Keys are generated deterministically from a shared secret (if provided) or auto-generated for open networks
*   **Discovery:** UDP Multicast/Broadcast is used to beacon presence on the local subnet. Nodes automatically perform handshakes upon discovery
*   **Protocol:** Messages are serialized using **Protocol Buffers (proto3)** to ensure strict typing and forward compatibility

## 3. Data Flow

### Copy Flow (Local -> Remote)
1.  **OS Event:** HAL receives a signal
2.  **Read:** Data is read from the OS buffer
3.  **Hash Check:** `Deduplicator` calculates xxHash. If `currentHash == lastWriteHash`, flow stops (Loop Protection)
4.  **Packetize:** Data is wrapped in a `DomainMessage` struct
5.  **Send:** The message is pushed to the `Channel` and broadcast to all active `Peer` connections

### Paste Flow (Remote -> Local)
1.  **Stream Accept:** QUIC transport accepts a new stream
2.  **Decode:** Protobuf decoder reconstructs the `DomainMessage`
3.  **File Handling:**
    *   *Text\Image:* Kept in memory
    *   *File:* Streamed to `internal/store` (disk)
4.  **Write:** The `Writer` component requests the OS to take ownership of the clipboard and sets the data
5.  **Mark:** The new data's hash is explicitly added to the `Deduplicator`'s ignore list to prevent re-broadcasting

## 4. Technology Stack

*   **Language:** Go 1.25+
*   **Transport:** QUIC, TLS 1.3
*   **Serialization:** Protobuf
*   **Hashing:** xxHash
*   **System Integration:**
    *   `deedles.dev/wl` (Wayland)
    *   `jezek/xgb` (X11)
    *   `ebitengine/purego` (macOS/FFI)
    *   `syscall` (Windows)