package fileexporter

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"text/template"

	"github.com/thomaspoignant/go-feature-flag/exporter"
)

type Exporter struct {
	// Format is the output format you want in your exported file.
	// Available format are JSON and CSV.
	// Default: JSON
	Format string

	// OutputDir is the location of the directory where to store the exported files
	// It should finish with a /
	// Default: the current directory
	OutputDir string

	// Filename is the name of your output file
	// You can use a templated config to define the name of your export files.
	// Available replacement are {{ .Hostname}}, {{ .Timestamp}} and {{ .Format}}
	// Default: "flag-variation-{{ .Hostname}}-{{ .Timestamp}}.{{ .Format}}"
	Filename string

	// CsvTemplate is used if your output format is CSV.
	// This field will be ignored if you are using another format than CSV.
	// You can decide which fields you want in your CSV line with a go-template syntax,
	// please check exporter/feature_event.go to see what are the fields available.
	// Default:
	// {{ .Kind}};{{ .ContextKind}};{{ .UserKey}};{{ .CreationDate}};{{ .Key}};{{ .Variation}};{{ .Value}};{{ .Default}}\n
	CsvTemplate string

	csvTemplate      *template.Template
	filenameTemplate *template.Template
	initTemplates    sync.Once
}

// Export is saving a collection of events in a file.
func (f *Exporter) Export(_ context.Context, _ *log.Logger, featureEvents []exporter.FeatureEvent) error {
	// Parse the template only once
	f.initTemplates.Do(func() {
		f.csvTemplate = exporter.ParseTemplate("csvFormat", f.CsvTemplate, exporter.DefaultCsvTemplate)
		f.filenameTemplate = exporter.ParseTemplate("filenameFormat", f.Filename, exporter.DefaultFilenameTemplate)
	})

	// Default format for the output
	if f.Format == "" {
		f.Format = "json"
	}

	// Get the filename
	filename, err := exporter.ComputeFilename(f.filenameTemplate, f.Format)
	if err != nil {
		return err
	}

	filePath := f.OutputDir + "/" + filename

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	for _, event := range featureEvents {
		var line []byte
		var err error

		// Convert the line in the right format
		switch strings.ToLower(f.Format) {
		case "csv":
			line, err = exporter.FormatEventInCSV(f.csvTemplate, event)
		case "json":
			line, err = exporter.FormatEventInJSON(event)
		default:
			line, err = exporter.FormatEventInJSON(event)
		}

		// Handle error and write line into the file
		if err != nil {
			return fmt.Errorf("impossible to format the event in %s: %v", f.Format, err)
		}
		_, errWrite := file.Write(line)
		if errWrite != nil {
			return fmt.Errorf("error while writing the export file: %v", err)
		}
	}
	return nil
}

// IsBulk return false if we should directly send the data as soon as it is produce
// and true if we collect the data to send them in bulk.
func (f *Exporter) IsBulk() bool {
	return true
}
