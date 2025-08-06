// Package processor - Export functions for TDOA results
package processor

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// ExportGeoJSON exports the TDOA results in GeoJSON format for web mapping
func (r *Result) ExportGeoJSON(filename string) error {
	// Create GeoJSON structure
	geojson := map[string]interface{}{
		"type": "FeatureCollection",
		"features": []map[string]interface{}{},
		"properties": map[string]interface{}{
			"title":           "TDOA Transmitter Location Analysis",
			"algorithm":       r.Algorithm,
			"frequency_mhz":   r.Frequency / 1e6,
			"confidence":      r.Confidence,
			"error_radius_m":  r.ErrorRadius,
			"processing_time": r.ProcessingTime.Format("2006-01-02T15:04:05Z"),
		},
	}

	features := []map[string]interface{}{}

	// Add estimated transmitter location as a point
	transmitterFeature := map[string]interface{}{
		"type": "Feature",
		"geometry": map[string]interface{}{
			"type":        "Point",
			"coordinates": []float64{r.Location.Longitude, r.Location.Latitude},
		},
		"properties": map[string]interface{}{
			"name":         "Estimated Transmitter Location",
			"type":         "transmitter",
			"confidence":   r.Confidence,
			"error_radius": r.ErrorRadius,
			"algorithm":    r.Algorithm,
		},
	}
	features = append(features, transmitterFeature)

	// Add confidence circle around transmitter location
	confidenceCircle := generateCircleFeature(r.Location, r.ErrorRadius, "confidence_area")
	features = append(features, confidenceCircle)

	// Add receiver locations
	for _, receiver := range r.ReceiverLocations {
		receiverFeature := map[string]interface{}{
			"type": "Feature",
			"geometry": map[string]interface{}{
				"type":        "Point",
				"coordinates": []float64{receiver.Location.Longitude, receiver.Location.Latitude},
			},
			"properties": map[string]interface{}{
				"name":     receiver.ID,
				"type":     "receiver",
				"filename": receiver.Filename,
				"snr_db":   receiver.SNR,
			},
		}
		features = append(features, receiverFeature)
	}

	// Add TDOA measurement lines
	for _, measurement := range r.TDOAMeasurements {
		// Find receiver locations
		var r1, r2 *ReceiverInfo
		for _, receiver := range r.ReceiverLocations {
			if receiver.ID == measurement.Receiver1ID {
				r1 = &receiver
			}
			if receiver.ID == measurement.Receiver2ID {
				r2 = &receiver
			}
		}

		if r1 != nil && r2 != nil {
			lineFeature := map[string]interface{}{
				"type": "Feature",
				"geometry": map[string]interface{}{
					"type": "LineString",
					"coordinates": [][]float64{
						{r1.Location.Longitude, r1.Location.Latitude},
						{r2.Location.Longitude, r2.Location.Latitude},
					},
				},
				"properties": map[string]interface{}{
					"name":             fmt.Sprintf("%s-%s TDOA", measurement.Receiver1ID, measurement.Receiver2ID),
					"type":             "tdoa_baseline",
					"time_diff_ns":     measurement.TimeDiff,
					"distance_diff_m":  measurement.DistanceDiff,
					"confidence":       measurement.Confidence,
					"correlation_peak": measurement.CorrelationPeak,
				},
			}
			features = append(features, lineFeature)
		}
	}

	// Add heatmap points if available
	if len(r.HeatmapPoints) > 0 {
		for _, point := range r.HeatmapPoints {
			if point.Probability > 0.1 { // Only include significant probability points
				heatmapFeature := map[string]interface{}{
					"type": "Feature",
					"geometry": map[string]interface{}{
						"type":        "Point",
						"coordinates": []float64{point.Location.Longitude, point.Location.Latitude},
					},
					"properties": map[string]interface{}{
						"type":        "heatmap",
						"probability": point.Probability,
					},
				}
				features = append(features, heatmapFeature)
			}
		}
	}

	geojson["features"] = features

	// Write to file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create GeoJSON file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(geojson); err != nil {
		return fmt.Errorf("failed to encode GeoJSON: %w", err)
	}

	return nil
}

// ExportKML exports the TDOA results in KML format for Google Earth
func (r *Result) ExportKML(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create KML file: %w", err)
	}
	defer file.Close()

	// Write KML header
	fmt.Fprintf(file, `<?xml version="1.0" encoding="UTF-8"?>
<kml xmlns="http://www.opengis.net/kml/2.2">
  <Document>
    <name>TDOA Transmitter Location Analysis</name>
    <description>Frequency: %.3f MHz, Algorithm: %s, Confidence: %.2f</description>
    
    <!-- Styles -->
    <Style id="transmitterStyle">
      <IconStyle>
        <Icon>
          <href>http://maps.google.com/mapfiles/kml/shapes/target.png</href>
        </Icon>
        <scale>1.5</scale>
        <color>ff0000ff</color>
      </IconStyle>
      <LabelStyle>
        <scale>1.2</scale>
      </LabelStyle>
    </Style>
    
    <Style id="receiverStyle">
      <IconStyle>
        <Icon>
          <href>http://maps.google.com/mapfiles/kml/shapes/placemark_circle.png</href>
        </Icon>
        <color>ff00ff00</color>
      </IconStyle>
    </Style>
    
    <Style id="confidenceStyle">
      <LineStyle>
        <color>7f0000ff</color>
        <width>2</width>
      </LineStyle>
      <PolyStyle>
        <color>3f0000ff</color>
      </PolyStyle>
    </Style>
    
    <Style id="baselineStyle">
      <LineStyle>
        <color>ff00ffff</color>
        <width>1</width>
      </LineStyle>
    </Style>
`, r.Frequency/1e6, r.Algorithm, r.Confidence)

	// Add estimated transmitter location
	fmt.Fprintf(file, `
    <Placemark>
      <name>Estimated Transmitter Location</name>
      <description>Confidence: %.2f, Error Radius: %.1f m, Algorithm: %s</description>
      <styleUrl>#transmitterStyle</styleUrl>
      <Point>
        <coordinates>%.8f,%.8f,%.1f</coordinates>
      </Point>
    </Placemark>
`, r.Confidence, r.ErrorRadius, r.Algorithm, r.Location.Longitude, r.Location.Latitude, r.Location.Altitude)

	// Add confidence circle
	fmt.Fprintf(file, `
    <Placemark>
      <name>Confidence Area</name>
      <description>%.1f meter radius confidence area</description>
      <styleUrl>#confidenceStyle</styleUrl>
      <Polygon>
        <outerBoundaryIs>
          <LinearRing>
            <coordinates>
`, r.ErrorRadius)

	// Generate circle points
	circlePoints := generateCirclePoints(r.Location, r.ErrorRadius, 36)
	for _, point := range circlePoints {
		fmt.Fprintf(file, "%.8f,%.8f,%.1f ", point.Longitude, point.Latitude, point.Altitude)
	}

	fmt.Fprintf(file, `
            </coordinates>
          </LinearRing>
        </outerBoundaryIs>
      </Polygon>
    </Placemark>
`)

	// Add receiver stations
	for _, receiver := range r.ReceiverLocations {
		fmt.Fprintf(file, `
    <Placemark>
      <name>%s</name>
      <description>SNR: %.1f dB, File: %s</description>
      <styleUrl>#receiverStyle</styleUrl>
      <Point>
        <coordinates>%.8f,%.8f,%.1f</coordinates>
      </Point>
    </Placemark>
`, receiver.ID, receiver.SNR, receiver.Filename, receiver.Location.Longitude, receiver.Location.Latitude, receiver.Location.Altitude)
	}

	// Add TDOA baseline measurements
	for _, measurement := range r.TDOAMeasurements {
		// Find receiver locations
		var r1, r2 *ReceiverInfo
		for _, receiver := range r.ReceiverLocations {
			if receiver.ID == measurement.Receiver1ID {
				r1 = &receiver
			}
			if receiver.ID == measurement.Receiver2ID {
				r2 = &receiver
			}
		}

		if r1 != nil && r2 != nil {
			fmt.Fprintf(file, `
    <Placemark>
      <name>%s-%s TDOA Baseline</name>
      <description>Time Diff: %.1f ns, Distance Diff: %.1f m, Confidence: %.3f</description>
      <styleUrl>#baselineStyle</styleUrl>
      <LineString>
        <coordinates>
          %.8f,%.8f,%.1f %.8f,%.8f,%.1f
        </coordinates>
      </LineString>
    </Placemark>
`, measurement.Receiver1ID, measurement.Receiver2ID, measurement.TimeDiff, measurement.DistanceDiff, measurement.Confidence,
				r1.Location.Longitude, r1.Location.Latitude, r1.Location.Altitude,
				r2.Location.Longitude, r2.Location.Latitude, r2.Location.Altitude)
		}
	}

	// Write KML footer
	fmt.Fprintf(file, `
  </Document>
</kml>
`)

	return nil
}

// ExportCSV exports the TDOA results in CSV format for spreadsheet analysis
func (r *Result) ExportCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write metadata header
	writer.Write([]string{"# TDOA Transmitter Location Analysis"})
	writer.Write([]string{"# Processing Time", r.ProcessingTime.Format("2006-01-02 15:04:05")})
	writer.Write([]string{"# Algorithm", r.Algorithm})
	writer.Write([]string{"# Frequency MHz", fmt.Sprintf("%.3f", r.Frequency/1e6)})
	writer.Write([]string{"# Estimated Location", fmt.Sprintf("%.8f,%.8f", r.Location.Latitude, r.Location.Longitude)})
	writer.Write([]string{"# Confidence", fmt.Sprintf("%.3f", r.Confidence)})
	writer.Write([]string{"# Error Radius m", fmt.Sprintf("%.1f", r.ErrorRadius)})
	writer.Write([]string{""}) // Empty line

	// Write receiver information
	writer.Write([]string{"# Receiver Stations"})
	writer.Write([]string{"Receiver_ID", "Latitude", "Longitude", "Altitude", "SNR_dB", "Filename"})
	for _, receiver := range r.ReceiverLocations {
		writer.Write([]string{
			receiver.ID,
			fmt.Sprintf("%.8f", receiver.Location.Latitude),
			fmt.Sprintf("%.8f", receiver.Location.Longitude),
			fmt.Sprintf("%.1f", receiver.Location.Altitude),
			fmt.Sprintf("%.1f", receiver.SNR),
			receiver.Filename,
		})
	}
	writer.Write([]string{""}) // Empty line

	// Write TDOA measurements
	writer.Write([]string{"# TDOA Measurements"})
	writer.Write([]string{"Receiver1_ID", "Receiver2_ID", "Time_Diff_ns", "Distance_Diff_m", "Confidence", "Correlation_Peak"})
	for _, measurement := range r.TDOAMeasurements {
		writer.Write([]string{
			measurement.Receiver1ID,
			measurement.Receiver2ID,
			fmt.Sprintf("%.1f", measurement.TimeDiff),
			fmt.Sprintf("%.1f", measurement.DistanceDiff),
			fmt.Sprintf("%.3f", measurement.Confidence),
			fmt.Sprintf("%.3f", measurement.CorrelationPeak),
		})
	}

	// Write heatmap points if available
	if len(r.HeatmapPoints) > 0 {
		writer.Write([]string{""}) // Empty line
		writer.Write([]string{"# Probability Heatmap Points"})
		writer.Write([]string{"Latitude", "Longitude", "Probability"})
		for _, point := range r.HeatmapPoints {
			writer.Write([]string{
				fmt.Sprintf("%.8f", point.Location.Latitude),
				fmt.Sprintf("%.8f", point.Location.Longitude),
				fmt.Sprintf("%.3f", point.Probability),
			})
		}
	}

	return nil
}

// generateCircleFeature creates a GeoJSON circle feature
func generateCircleFeature(center Location, radius float64, featureType string) map[string]interface{} {
	points := generateCirclePoints(center, radius, 64)
	
	coordinates := make([][]float64, len(points)+1) // +1 to close the polygon
	for i, point := range points {
		coordinates[i] = []float64{point.Longitude, point.Latitude}
	}
	// Close the polygon by repeating the first point
	coordinates[len(points)] = []float64{points[0].Longitude, points[0].Latitude}

	return map[string]interface{}{
		"type": "Feature",
		"geometry": map[string]interface{}{
			"type":        "Polygon",
			"coordinates": [][][]float64{coordinates},
		},
		"properties": map[string]interface{}{
			"name":   "Confidence Area",
			"type":   featureType,
			"radius": radius,
		},
	}
}

// generateCirclePoints generates points around a circle for a given center and radius
func generateCirclePoints(center Location, radiusMeters float64, numPoints int) []Location {
	points := make([]Location, numPoints)
	
	// Convert radius from meters to degrees (approximate)
	latRadiusDeg := radiusMeters / 111000.0 // Approximate meters per degree latitude
	lonRadiusDeg := radiusMeters / (111000.0 * math.Cos(center.Latitude * math.Pi / 180))
	
	for i := 0; i < numPoints; i++ {
		angle := 2 * math.Pi * float64(i) / float64(numPoints)
		
		points[i] = Location{
			Latitude:  center.Latitude + latRadiusDeg * math.Sin(angle),
			Longitude: center.Longitude + lonRadiusDeg * math.Cos(angle),
			Altitude:  center.Altitude,
		}
	}
	
	return points
}