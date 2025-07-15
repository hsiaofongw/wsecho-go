# Demo Latency Tester and Online Visitors Tracking

This project demonstrates how to count online visitors and measure round-trip latency.

## Highlights

- Single file implementation (one file for client, one for server)
- Dark mode supported

## Quick Start

### Prerequisites

- Go 1.16 or later
- Python 3.x (for static file server)

### Running the Demo

1. **Start the WebSocket Server**:

   ```bash
   go run .
   ```

   The server will start on port 8082.

2. **Start the Static Web Server**:

   ```bash
   python -m http.server 8081
   ```

   The web interface will be available on port 8081.

3. **Access the Demo**:
   Open your browser and navigate to:
   ```
   http://localhost:8081
   ```

### Multiple Client Testing

To test with multiple clients:

1. Open multiple browser windows to `http://localhost:8081`
2. Each client will get a unique session ID
3. Watch the online count increase and latency measurements

## Development

### Project Structure

```
go-echo/
├── main.go           # WebSocket server implementation
├── static/
│   └── index.html    # Web client interface
├── go.mod            # Go module dependencies
└── go.sum            # Dependency checksums
```

### Building

```bash
# Build the server
go build -o echo-server .

# Run the built binary
./echo-server
```

## Troubleshooting

### Common Issues

1. **Port Already in Use**:

   - Ensure ports 8081 and 8082 are available
   - Check for existing processes using these ports

2. **WebSocket Connection Failed**:

   - Verify the server is running on port 8082
   - Check browser console for connection errors

3. **Session Issues**:
   - Clear browser cookies if session problems occur
   - Restart the server to reset all sessions

## Contributing

Feel free to submit issues, feature requests, or pull requests to improve this demo project.
