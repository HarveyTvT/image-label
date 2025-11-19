package main

import (
	"archive/zip"
	"bufio"
	"embed"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

//go:embed templates/*
var templateFS embed.FS

const (
	imagesDir    = "./images"
	itemsPerPage = 10
	choiceFile   = "./labels.txt"
)

type PageData struct {
	Images      []string
	CurrentPage int
	TotalPages  int
	Labels      []LabelChoice
}

type LabelChoice struct {
	Index int
	Text  string
}

var labelChoices []string

func main() {
	// Create images directory if it doesn't exist
	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		log.Fatalf("Failed to create images directory: %v", err)
	}

	// Load label choices from file
	var err error
	labelChoices, err = loadLabelChoices(choiceFile)
	if err != nil {
		log.Fatalf("Failed to load label choices: %v", err)
	}
	log.Printf("Loaded %d label choices", len(labelChoices))

	// Clean label subfolders from previous runs
	if err := cleanLabelSubfolders(); err != nil {
		log.Printf("Warning: Failed to clean label subfolders: %v", err)
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\nReceived interrupt signal, creating result.zip...")
		if err := createResultZip(); err != nil {
			log.Printf("Error creating result.zip: %v", err)
		} else {
			log.Println("Successfully created result.zip")
		}
		os.Exit(0)
	}()

	// Routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/label", labelHandler)
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(imagesDir))))

	port := ":18081"
	fmt.Printf("Server starting on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

func loadLabelChoices(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var labels []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			labels = append(labels, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(labels) == 0 {
		return nil, fmt.Errorf("no labels found in %s", filename)
	}

	return labels, nil
}

func cleanLabelSubfolders() error {
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Remove all subdirectories in the images folder
			dirPath := filepath.Join(imagesDir, entry.Name())
			if err := os.RemoveAll(dirPath); err != nil {
				log.Printf("Failed to remove directory %s: %v", dirPath, err)
			} else {
				log.Printf("Cleaned label subfolder: %s", entry.Name())
			}
		}
	}

	return nil
}

func createResultZip() error {
	// Create the zip file
	zipFile, err := os.Create("result.zip")
	if err != nil {
		return fmt.Errorf("failed to create result.zip: %v", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Read all entries in the images directory
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		return fmt.Errorf("failed to read images directory: %v", err)
	}

	// Add each label subfolder to the zip
	for _, entry := range entries {
		if entry.IsDir() {
			labelDir := filepath.Join(imagesDir, entry.Name())

			// Walk through the label subfolder
			err := filepath.Walk(labelDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// Skip directories themselves
				if info.IsDir() {
					return nil
				}

				// Get the relative path for the zip entry
				relPath, err := filepath.Rel(imagesDir, path)
				if err != nil {
					return err
				}

				// Create a zip entry
				zipEntry, err := zipWriter.Create(relPath)
				if err != nil {
					return fmt.Errorf("failed to create zip entry: %v", err)
				}

				// Open the file to be added
				file, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("failed to open file %s: %v", path, err)
				}
				defer file.Close()

				// Copy file contents to zip entry
				if _, err := io.Copy(zipEntry, file); err != nil {
					return fmt.Errorf("failed to write file to zip: %v", err)
				}

				log.Printf("Added to zip: %s", relPath)
				return nil
			})
			if err != nil {
				log.Printf("Error processing directory %s: %v", labelDir, err)
			}
		}
	}

	return nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Get page number from query parameter
	pageStr := r.URL.Query().Get("page")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	// Get all images from the images directory (excluding subdirectories)
	images, err := getImages()
	if err != nil {
		http.Error(w, "Failed to read images", http.StatusInternalServerError)
		log.Printf("Error reading images: %v", err)
		return
	}

	// Calculate pagination
	totalImages := len(images)
	totalPages := (totalImages + itemsPerPage - 1) / itemsPerPage
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Get images for current page
	startIdx := (page - 1) * itemsPerPage
	endIdx := startIdx + itemsPerPage
	if endIdx > totalImages {
		endIdx = totalImages
	}

	var pageImages []string
	if startIdx < totalImages {
		pageImages = images[startIdx:endIdx]
	}

	// Prepare label choices with indices
	labels := make([]LabelChoice, len(labelChoices))
	for i, text := range labelChoices {
		labels[i] = LabelChoice{
			Index: i + 1,
			Text:  text,
		}
	}

	// Prepare template data
	data := PageData{
		Images:      pageImages,
		CurrentPage: page,
		TotalPages:  totalPages,
		Labels:      labels,
	}

	// Parse and execute template with custom functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
	}

	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFS(templateFS, "templates/index.html")
	if err != nil {
		http.Error(w, "Failed to load template", http.StatusInternalServerError)
		log.Printf("Error parsing template: %v", err)
		return
	}

	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

func labelHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	imageName := r.FormValue("image")
	labelStr := r.FormValue("label")

	if imageName == "" || labelStr == "" {
		http.Error(w, "Missing image or label", http.StatusBadRequest)
		return
	}

	labelIndex, err := strconv.Atoi(labelStr)
	if err != nil || labelIndex < 1 || labelIndex > len(labelChoices) {
		http.Error(w, "Invalid label", http.StatusBadRequest)
		return
	}

	// Get the label text from the index (1-based)
	labelName := labelChoices[labelIndex-1]

	// Source and destination paths
	srcPath := filepath.Join(imagesDir, imageName)
	dstDir := filepath.Join(imagesDir, labelName)
	dstPath := filepath.Join(dstDir, imageName)

	// Create label subfolder if it doesn't exist
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		http.Error(w, "Failed to create label directory", http.StatusInternalServerError)
		log.Printf("Error creating label directory: %v", err)
		return
	}

	// Move the image to the label subfolder
	if err := moveFile(srcPath, dstPath); err != nil {
		http.Error(w, "Failed to move image", http.StatusInternalServerError)
		log.Printf("Error moving image: %v", err)
		return
	}

	log.Printf("Labeled image %s with label '%s' (%d)", imageName, labelName, labelIndex)

	// Return success response (for AJAX)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func getImages() ([]string, error) {
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		return nil, err
	}

	var images []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Filter for common image extensions
		ext := filepath.Ext(entry.Name())
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp":
			images = append(images, entry.Name())
		}
	}

	// Sort images by name
	sort.Strings(images)
	return images, nil
}

func moveFile(src, dst string) error {
	// Try to rename first (fastest if on same filesystem)
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// If rename fails (different filesystem), copy then delete
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Remove source file after successful copy
	return os.Remove(src)
}
