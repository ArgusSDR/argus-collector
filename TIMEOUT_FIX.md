# Collection Timeout Fix

## Problem
The argus-collector would not stop collection after the specified duration was reached. This occurred because:

1. **Blocking RTL-SDR Calls**: The `ReadSync()` call in RTL-SDR collection could block indefinitely
2. **Inadequate Timeout Handling**: The collector would wait indefinitely for the RTL-SDR goroutine to complete
3. **Poor Goroutine Coordination**: No proper mechanism to signal collection completion or cancellation

## Solution

### 1. Context-Based Timeout in RTL-SDR
- Added context with timeout to both real and stub RTL-SDR implementations
- Collection loop now checks context cancellation between read operations
- Ensures collection stops within the specified duration

### 2. Improved Collector Timeout Logic
- Added multiple layers of timeout protection:
  - Data collection timeout: `duration + 5 seconds`
  - Maximum wait timeout: `duration + 10 seconds`
  - Goroutine cleanup timeout: `5 seconds`

### 3. Better Channel Management
- RTL-SDR goroutine now closes the samples channel when complete
- Collection handler detects closed channels properly
- Added proper error handling for channel closure

### 4. Enhanced Stub Implementation
- Stub now properly simulates timing behavior
- Supports all RTL-SDR configuration methods for testing
- Uses proper context-based timeout handling

## Code Changes

### RTL-SDR Interface (`internal/rtlsdr/rtlsdr.go`)
```go
// Added context with timeout
ctx, cancel := context.WithTimeout(context.Background(), duration)
defer cancel()

// Check context in collection loop
select {
case <-ctx.Done():
    break // Stop collection
default:
    // Continue reading
}
```

### Stub Implementation (`internal/rtlsdr/rtlsdr_stub.go`)
```go
// Proper timing simulation
select {
case <-ctx.Done():
    // Duration expired - normal completion
case <-time.After(duration + time.Second):
    return fmt.Errorf("stub collection timeout exceeded")
}
```

### Collector Logic (`internal/collector/collector.go`)
```go
// Multiple timeout layers
select {
case err := <-done:
    // Collection completed
case <-time.After(c.config.Collection.Duration + 10*time.Second):
    return fmt.Errorf("collection timeout - exceeded maximum wait time")
}

// Goroutine cleanup with timeout
select {
case <-waitDone:
    // Normal completion
case <-time.After(5 * time.Second):
    fmt.Printf("Warning: RTL-SDR collection goroutine did not complete in time\n")
}
```

## Testing

### Manual Testing
```bash
# Test 5-second collection should complete in ~5 seconds
time ./argus-collector --duration=5s --gps-mode=manual --latitude=35.533 --longitude=-97.621

# Should complete in approximately 5 seconds (plus startup/shutdown time)
```

### Expected Behavior
- Collection starts immediately after GPS fix (or uses manual coordinates)
- RTL-SDR data collection runs for exactly the specified duration
- Application exits promptly after collection completion
- No hanging processes or indefinite waits

### Error Handling
- If RTL-SDR hardware fails: timeout after `duration + 5 seconds`
- If file writing fails: immediate error return
- If goroutine hangs: warning message, but application exits

## Troubleshooting

### Collection Still Hangs
1. Check RTL-SDR hardware connection
2. Verify permissions (plugdev group)
3. Try stub mode: `make build-stub && ./argus-collector --duration=5s`

### Timeout Errors
- `collection timeout - no data received from RTL-SDR`: Hardware issue
- `collection timeout - exceeded maximum wait time`: System overload
- `RTL-SDR collection goroutine did not complete in time`: Background cleanup issue (safe to ignore)

### Performance Issues
- Use shorter durations for testing: `--duration=2s`
- Check system resources: `top`, `free -h`
- Verify disk space for output files: `df -h ./data/`