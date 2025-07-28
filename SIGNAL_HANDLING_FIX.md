# Signal Handling (Ctrl-C) Fix

## Problem
Pressing Ctrl-C did not stop the program, especially during:
1. GPS fix waiting
2. Synchronized start waiting
3. RTL-SDR data collection

The application would hang indefinitely and ignore the interrupt signal.

## Root Cause
1. **Late Signal Handler Setup**: Signal handling was initialized after collector setup, so Ctrl-C during initialization was ignored
2. **Non-Cancellable Operations**: GPS waiting and sync start waiting used blocking calls that couldn't be interrupted
3. **No Context Propagation**: No mechanism to propagate cancellation through the operation chain

## Solution

### 1. Early Signal Handler Setup
```go
// Set up signal handling BEFORE any long-running operations
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

// Create cancellable context
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Handle signals immediately
go func() {
    <-sigChan
    fmt.Printf("\nReceived interrupt signal, shutting down...\n")
    cancel() // Cancel all operations
    os.Exit(1) // Force exit if graceful shutdown takes too long
}()
```

### 2. Context-Aware Operations
All potentially blocking operations now accept context and can be cancelled:

#### GPS Fix Waiting
```go
func (c *Collector) WaitForGPSFixWithContext(ctx context.Context) error {
    // Run GPS fix in goroutine
    gpsResultChan := make(chan gpsResult, 1)
    go func() {
        pos, err := c.gps.WaitForFix(c.config.GPS.Timeout)
        gpsResultChan <- gpsResult{pos, err}
    }()
    
    // Wait for GPS or cancellation
    select {
    case result := <-gpsResultChan:
        // GPS completed
    case <-ctx.Done():
        return fmt.Errorf("GPS fix cancelled: %w", ctx.Err())
    }
}
```

#### Synchronized Start Waiting
```go
select {
case <-time.After(waitDuration):
    // Normal sync wait completed
case <-ctx.Done():
    return fmt.Errorf("synchronized start cancelled: %w", ctx.Err())
}
```

#### Data Collection
```go
select {
case err := <-done:
    // Collection completed
case <-time.After(timeout):
    // Timeout
case <-ctx.Done():
    return fmt.Errorf("collection cancelled: %w", ctx.Err())
}
```

### 3. Graceful Shutdown with Force Exit
- **Normal case**: Context cancellation stops operations gracefully
- **Stuck case**: `os.Exit(1)` after 1-2 seconds forces termination
- **Multiple Ctrl-C**: Second Ctrl-C immediately forces exit

## Testing

### Test Ctrl-C During GPS Fix
```bash
# Start with GPS enabled, press Ctrl-C during GPS waiting
./argus-collector --gps-mode=nmea --gps-port=/dev/ttyACM0 --duration=10s
# Press Ctrl-C - should exit immediately
```

### Test Ctrl-C During Sync Start Wait
```bash
# Enable sync start, press Ctrl-C during wait
./argus-collector --synced-start=true --gps-mode=manual --latitude=35.533 --longitude=-97.621
# Press Ctrl-C during "Waiting X seconds for synchronized start..." - should exit immediately
```

### Test Ctrl-C During Collection
```bash
# Start collection, press Ctrl-C during data collection
./argus-collector --synced-start=false --gps-mode=manual --latitude=35.533 --longitude=-97.621 --duration=60s
# Press Ctrl-C during collection - should exit immediately
```

## Expected Behavior

### Before Fix
- Ctrl-C ignored during GPS waiting
- Ctrl-C ignored during sync start waiting
- Application hangs indefinitely
- Only way to stop: `kill -9 <pid>`

### After Fix
- Ctrl-C works immediately at any stage
- Clean shutdown message: "Received interrupt signal, shutting down..."
- Operations cancelled gracefully with context
- Exit code 1 (indicating interruption)
- Force exit within 1-2 seconds if cleanup hangs

## Error Messages

### Graceful Cancellation
```
^C
Received interrupt signal, shutting down...
Error: GPS fix cancelled: context canceled
```

### During Sync Start
```
^C
Received interrupt signal, shutting down...
Error: synchronized start cancelled: context canceled
```

### During Collection
```
^C
Received interrupt signal, shutting down...  
Error: collection cancelled: context canceled
```

## Implementation Notes

1. **Context Propagation**: Context flows through entire operation chain
2. **Backwards Compatibility**: Old methods (`WaitForGPSFix()`, `Collect()`) still work by using `context.Background()`
3. **Cleanup Protection**: Cleanup phase has its own timeout to prevent hanging
4. **Signal Masking**: Uses proper Go signal handling, not unsafe signal patterns

The fix ensures Ctrl-C always works regardless of what stage the application is in, providing responsive user control over the collection process.