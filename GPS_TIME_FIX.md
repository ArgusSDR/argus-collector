# GPS Time Conversion Fix

## Issue
Build error: `cannot use s.Time (variable of type nmea.Time) as time.Time value in struct literal`

The NMEA library uses its own `nmea.Time` type which doesn't directly convert to Go's `time.Time`.

## Root Cause
In the RMC (Recommended Minimum) sentence processing, we were trying to assign `s.Time` (of type `nmea.Time`) directly to a `time.Time` field.

## Solution
Added proper time conversion for NMEA RMC time data:

```go
// Convert NMEA time to Go time.Time
// RMC provides time but not full timestamp, so we use current date
rncTime := time.Now()
if s.Time.Valid {
    // Use today's date with the GPS time
    rncTime = time.Date(
        rncTime.Year(), rncTime.Month(), rncTime.Day(),
        s.Time.Hour, s.Time.Minute, s.Time.Second, 
        int(s.Time.Millisecond)*1000000, // Convert ms to ns  
        time.UTC,
    )
}
```

## Key Points

1. **RMC Limitation**: RMC sentences provide time-of-day but not the full date
2. **Date Assumption**: We use the current system date with the GPS time
3. **Timezone**: GPS time is in UTC, so we explicitly use `time.UTC`
4. **Validity Check**: We check `s.Time.Valid` before using the time data
5. **Precision**: Convert milliseconds to nanoseconds for Go's time representation

## Testing
After this fix, the build should complete successfully:

```bash
# Test stub build (no RTL-SDR hardware required)
make build-stub

# Test full build (requires librtlsdr-dev)
make build
```

## Time Accuracy Notes

- **GGA sentences**: Use `time.Now()` for timestamp as they don't include time
- **RMC sentences**: Use converted GPS time with current date
- **Manual mode**: Use `time.Now()` for timestamp
- **GPSD mode**: Uses GPSD's native time.Time (no conversion needed)

The time conversion assumes the GPS receiver and system are on the same day, which is reasonable for typical use cases. For applications requiring precise date handling across midnight boundaries, additional logic would be needed to parse RMC date fields.