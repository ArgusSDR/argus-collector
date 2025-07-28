# Synchronized Start Fix

## Issue
The `--synced-start=false` flag was not working - the application still waited for synchronized start timing even when explicitly disabled.

## Root Cause
The configuration loading precedence was not properly handling command line flags that override config file values. Viper's automatic unmarshaling wasn't respecting the explicit `--synced-start=false` flag when the config file also specified `synced_start: true`.

## Solution

### 1. Added Explicit Flag Override Check
```go
// Handle explicit command line flags that should override config file
if rootCmd.Flags().Changed("synced-start") {
    cfg.Collection.SyncedStart = syncedStart
}
```

This ensures that when `--synced-start=false` is explicitly provided, it overrides any config file setting.

### 2. Enhanced Debug Output
```go
fmt.Printf("Synchronized Start: %t\n", cfg.Collection.SyncedStart)

if c.config.Collection.SyncedStart {
    // ... synchronized start logic
} else {
    fmt.Printf("Synchronized start disabled - starting immediately\n")
    startTime = time.Now()
}
```

Now the application clearly shows whether synchronized start is enabled or disabled.

## Configuration Precedence

The correct order is now:
1. **Default configuration** (`SyncedStart: true`)
2. **Config file** (`synced_start: true`)  
3. **Command line flags** (`--synced-start=false` overrides all)

## Testing

### Test Immediate Start
```bash
# Should start immediately without waiting
./argus-collector --synced-start=false --gps-mode=manual --latitude=35.533 --longitude=-97.621 --duration=5s
```

Expected output:
```
Argus Collector starting...
Frequency: 433.92 MHz
Duration: 5s
Output: ./data
Synchronized Start: false
GPS: MANUAL MODE (using fixed coordinates)
Location: 35.53300000°, -97.62100000° (0.0 m)
Synchronized start disabled - starting immediately
Starting collection (ID: argus_1234567890)
```

### Test Synchronized Start (Default)
```bash
# Should wait for synchronized start time
./argus-collector --gps-mode=manual --latitude=35.533 --longitude=-97.621 --duration=5s
```

Expected output:
```
Argus Collector starting...
Frequency: 433.92 MHz
Duration: 5s
Output: ./data
Synchronized Start: true
GPS: MANUAL MODE (using fixed coordinates)
Location: 35.53300000°, -97.62100000° (0.0 m)
Synchronized start enabled - waiting until: 14:25:23.000
Waiting 12.345 seconds for synchronized start...
Starting collection (ID: argus_1234567890)
```

### Verify Flag Recognition
The key indicator is the "Synchronized Start: false" message in the startup output when using `--synced-start=false`.

## Synchronized Start Algorithm

When enabled (`--synced-start=true` or default):
1. Calculate sync point: `(current_epoch + 5) % 100`
2. Find next minute where seconds equals sync point
3. Wait until that exact time
4. Start collection

When disabled (`--synced-start=false`):
1. Start collection immediately with `time.Now()`
2. No waiting or timing calculations

## Troubleshooting

### Still Waiting Despite --synced-start=false
- Check startup output for "Synchronized Start: false"
- If it shows "true", the flag override logic failed
- Try without config file: `./argus-collector --config=/dev/null --synced-start=false`

### Unexpected Start Times  
- With `--synced-start=false`: Should start within 1-2 seconds
- With synchronized start: May wait up to 60 seconds for next sync point
- GPS fix time adds to startup delay regardless of sync setting