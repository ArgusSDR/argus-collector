package filewriter

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"
)

type Metadata struct {
	Frequency         uint64
	SampleRate        uint32
	CollectionTime    time.Time
	GPSLocation       GPSLocation
	GPSTimestamp      time.Time
	DeviceInfo        string
	FileFormatVersion uint16
	CollectionID      string
}

type GPSLocation struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
}

type Writer struct{}

func NewWriter() *Writer {
	return &Writer{}
}

func (w *Writer) WriteFile(filename string, metadata Metadata, samples []complex64) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if err := w.writeHeader(file, metadata, uint32(len(samples))); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	if err := w.writeSamples(file, samples); err != nil {
		return fmt.Errorf("failed to write samples: %w", err)
	}

	return nil
}

func (w *Writer) writeHeader(file *os.File, metadata Metadata, sampleCount uint32) error {
	if _, err := file.WriteString("ARGUS"); err != nil {
		return err
	}

	if err := binary.Write(file, binary.LittleEndian, metadata.FileFormatVersion); err != nil {
		return err
	}

	if err := binary.Write(file, binary.LittleEndian, metadata.Frequency); err != nil {
		return err
	}

	if err := binary.Write(file, binary.LittleEndian, metadata.SampleRate); err != nil {
		return err
	}

	collectionTimeUnix := metadata.CollectionTime.Unix()
	collectionTimeNano := metadata.CollectionTime.Nanosecond()
	if err := binary.Write(file, binary.LittleEndian, int64(collectionTimeUnix)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, int32(collectionTimeNano)); err != nil {
		return err
	}

	if err := binary.Write(file, binary.LittleEndian, metadata.GPSLocation.Latitude); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, metadata.GPSLocation.Longitude); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, metadata.GPSLocation.Altitude); err != nil {
		return err
	}

	gpsTimeUnix := metadata.GPSTimestamp.Unix()
	gpsTimeNano := metadata.GPSTimestamp.Nanosecond()
	if err := binary.Write(file, binary.LittleEndian, int64(gpsTimeUnix)); err != nil {
		return err
	}
	if err := binary.Write(file, binary.LittleEndian, int32(gpsTimeNano)); err != nil {
		return err
	}

	deviceInfoBytes := []byte(metadata.DeviceInfo)
	if len(deviceInfoBytes) > 255 {
		deviceInfoBytes = deviceInfoBytes[:255]
	}
	deviceInfoLen := uint8(len(deviceInfoBytes))
	if err := binary.Write(file, binary.LittleEndian, deviceInfoLen); err != nil {
		return err
	}
	if _, err := file.Write(deviceInfoBytes); err != nil {
		return err
	}

	collectionIDBytes := []byte(metadata.CollectionID)
	if len(collectionIDBytes) > 255 {
		collectionIDBytes = collectionIDBytes[:255]
	}
	collectionIDLen := uint8(len(collectionIDBytes))
	if err := binary.Write(file, binary.LittleEndian, collectionIDLen); err != nil {
		return err
	}
	if _, err := file.Write(collectionIDBytes); err != nil {
		return err
	}

	if err := binary.Write(file, binary.LittleEndian, sampleCount); err != nil {
		return err
	}

	return nil
}

func (w *Writer) writeSamples(file *os.File, samples []complex64) error {
	for _, sample := range samples {
		if err := binary.Write(file, binary.LittleEndian, real(sample)); err != nil {
			return err
		}
		if err := binary.Write(file, binary.LittleEndian, imag(sample)); err != nil {
			return err
		}
	}
	return nil
}

// ReadFile reads the complete file including all sample data
func ReadFile(filename string) (*Metadata, []complex64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	magic := make([]byte, 5)
	if _, err := file.Read(magic); err != nil {
		return nil, nil, fmt.Errorf("failed to read magic: %w", err)
	}
	if string(magic) != "ARGUS" {
		return nil, nil, fmt.Errorf("invalid file format")
	}

	var metadata Metadata
	if err := binary.Read(file, binary.LittleEndian, &metadata.FileFormatVersion); err != nil {
		return nil, nil, err
	}

	if err := binary.Read(file, binary.LittleEndian, &metadata.Frequency); err != nil {
		return nil, nil, err
	}

	if err := binary.Read(file, binary.LittleEndian, &metadata.SampleRate); err != nil {
		return nil, nil, err
	}

	var collectionTimeUnix int64
	var collectionTimeNano int32
	if err := binary.Read(file, binary.LittleEndian, &collectionTimeUnix); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &collectionTimeNano); err != nil {
		return nil, nil, err
	}
	metadata.CollectionTime = time.Unix(collectionTimeUnix, int64(collectionTimeNano))

	if err := binary.Read(file, binary.LittleEndian, &metadata.GPSLocation.Latitude); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &metadata.GPSLocation.Longitude); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &metadata.GPSLocation.Altitude); err != nil {
		return nil, nil, err
	}

	var gpsTimeUnix int64
	var gpsTimeNano int32
	if err := binary.Read(file, binary.LittleEndian, &gpsTimeUnix); err != nil {
		return nil, nil, err
	}
	if err := binary.Read(file, binary.LittleEndian, &gpsTimeNano); err != nil {
		return nil, nil, err
	}
	metadata.GPSTimestamp = time.Unix(gpsTimeUnix, int64(gpsTimeNano))

	var deviceInfoLen uint8
	if err := binary.Read(file, binary.LittleEndian, &deviceInfoLen); err != nil {
		return nil, nil, err
	}
	deviceInfoBytes := make([]byte, deviceInfoLen)
	if _, err := file.Read(deviceInfoBytes); err != nil {
		return nil, nil, err
	}
	metadata.DeviceInfo = string(deviceInfoBytes)

	var collectionIDLen uint8
	if err := binary.Read(file, binary.LittleEndian, &collectionIDLen); err != nil {
		return nil, nil, err
	}
	collectionIDBytes := make([]byte, collectionIDLen)
	if _, err := file.Read(collectionIDBytes); err != nil {
		return nil, nil, err
	}
	metadata.CollectionID = string(collectionIDBytes)

	var sampleCount uint32
	if err := binary.Read(file, binary.LittleEndian, &sampleCount); err != nil {
		return nil, nil, err
	}

	samples := make([]complex64, sampleCount)
	for i := uint32(0); i < sampleCount; i++ {
		var real, imag float32
		if err := binary.Read(file, binary.LittleEndian, &real); err != nil {
			return nil, nil, err
		}
		if err := binary.Read(file, binary.LittleEndian, &imag); err != nil {
			return nil, nil, err
		}
		samples[i] = complex(real, imag)
	}

	return &metadata, samples, nil
}

// ReadMetadata reads only the metadata header without loading sample data
func ReadMetadata(filename string) (*Metadata, uint32, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read magic header
	magic := make([]byte, 5)
	if _, err := file.Read(magic); err != nil {
		return nil, 0, fmt.Errorf("failed to read magic: %w", err)
	}
	if string(magic) != "ARGUS" {
		return nil, 0, fmt.Errorf("invalid file format")
	}

	var metadata Metadata

	// Read metadata fields in order
	if err := binary.Read(file, binary.LittleEndian, &metadata.FileFormatVersion); err != nil {
		return nil, 0, err
	}

	if err := binary.Read(file, binary.LittleEndian, &metadata.Frequency); err != nil {
		return nil, 0, err
	}

	if err := binary.Read(file, binary.LittleEndian, &metadata.SampleRate); err != nil {
		return nil, 0, err
	}

	var collectionTimeUnix int64
	var collectionTimeNano int32
	if err := binary.Read(file, binary.LittleEndian, &collectionTimeUnix); err != nil {
		return nil, 0, err
	}
	if err := binary.Read(file, binary.LittleEndian, &collectionTimeNano); err != nil {
		return nil, 0, err
	}
	metadata.CollectionTime = time.Unix(collectionTimeUnix, int64(collectionTimeNano))

	if err := binary.Read(file, binary.LittleEndian, &metadata.GPSLocation.Latitude); err != nil {
		return nil, 0, err
	}
	if err := binary.Read(file, binary.LittleEndian, &metadata.GPSLocation.Longitude); err != nil {
		return nil, 0, err
	}
	if err := binary.Read(file, binary.LittleEndian, &metadata.GPSLocation.Altitude); err != nil {
		return nil, 0, err
	}

	var gpsTimeUnix int64
	var gpsTimeNano int32
	if err := binary.Read(file, binary.LittleEndian, &gpsTimeUnix); err != nil {
		return nil, 0, err
	}
	if err := binary.Read(file, binary.LittleEndian, &gpsTimeNano); err != nil {
		return nil, 0, err
	}
	metadata.GPSTimestamp = time.Unix(gpsTimeUnix, int64(gpsTimeNano))

	var deviceInfoLen uint8
	if err := binary.Read(file, binary.LittleEndian, &deviceInfoLen); err != nil {
		return nil, 0, err
	}
	deviceInfoBytes := make([]byte, deviceInfoLen)
	if _, err := file.Read(deviceInfoBytes); err != nil {
		return nil, 0, err
	}
	metadata.DeviceInfo = string(deviceInfoBytes)

	var collectionIDLen uint8
	if err := binary.Read(file, binary.LittleEndian, &collectionIDLen); err != nil {
		return nil, 0, err
	}
	collectionIDBytes := make([]byte, collectionIDLen)
	if _, err := file.Read(collectionIDBytes); err != nil {
		return nil, 0, err
	}
	metadata.CollectionID = string(collectionIDBytes)

	var sampleCount uint32
	if err := binary.Read(file, binary.LittleEndian, &sampleCount); err != nil {
		return nil, 0, err
	}

	return &metadata, sampleCount, nil
}

// ReadSamples reads only a specified number of samples from the file
func ReadSamples(filename string, offset, count uint32) ([]complex64, error) {
	// Use the existing ReadFile function but limit processing
	_, allSamples, err := ReadFile(filename)
	if err != nil {
		return nil, err
	}

	sampleCount := uint32(len(allSamples))
	if offset >= sampleCount {
		return nil, fmt.Errorf("offset %d exceeds sample count %d", offset, sampleCount)
	}

	// Limit count to available samples
	if offset+count > sampleCount {
		count = sampleCount - offset
	}

	// Return the requested slice
	return allSamples[offset : offset+count], nil
}
