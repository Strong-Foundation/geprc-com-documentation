package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"golang.org/x/net/html"
)

func main() {
	outputDirectory := "PDFs/" // Directory where downloaded PDF files will be saved
	if !directoryExists(outputDirectory) { // Check if the directory already exists
		createDirectory(outputDirectory, 0o755) // Create the directory with full read, write, and execute permissions (rwxr-xr-x)
	}
	outputDirZIP := "ZIPs/"             // Directory to store downloaded ZIPs
	if !directoryExists(outputDirZIP) { // Check if directory exists
		createDirectory(outputDirZIP, 0o755) // Create directory with read-write-execute permissions
	}
	urls := []string{
		"https://geprc.com/downloads/cinebot30/",
		"https://geprc.com/downloads/cinelog20/",
		"https://geprc.com/downloads/cinelog25/",
		"https://geprc.com/downloads/cinelog25-v2/",
		"https://geprc.com/downloads/cinelog30/",
		"https://geprc.com/downloads/cinelog30-v2/",
		"https://geprc.com/downloads/cinelog30-v3/",
		"https://geprc.com/downloads/cinelog35/",
		"https://geprc.com/downloads/cinelog35-performance/",
		"https://geprc.com/downloads/cinelog35-v2/",
		"https://geprc.com/downloads/cinepro/",
		"https://geprc.com/downloads/crocodileseries/",
		"https://geprc.com/downloads/crocodile5-baby-lr/",
		"https://geprc.com/downloads/crocodile75-v3/",
		"https://geprc.com/downloads/crown/",
		"https://geprc.com/downloads/darkstar16/",
		"https://geprc.com/downloads/darkstar20/",
		"https://geprc.com/downloads/domain-3-6/",
		"https://geprc.com/downloads/domain-4-2/",
		"https://geprc.com/downloads/mark4-7-inch/",
		"https://geprc.com/downloads/mark4-series/",
		"https://geprc.com/downloads/mark5/",
		"https://geprc.com/downloads/mk5d-lr7/",
		"https://geprc.com/downloads/moz7/",
		"https://geprc.com/downloads/moz7-v2/",
		"https://geprc.com/downloads/phantom/",
		"https://geprc.com/downloads/racer/",
		"https://geprc.com/downloads/rocket/",
		"https://geprc.com/downloads/smart16/",
		"https://geprc.com/downloads/smart35/",
		"https://geprc.com/downloads/thinking-p16/",
		"https://geprc.com/downloads/tinygo/",
		"https://geprc.com/downloads/vapord/",
		"https://geprc.com/downloads/",
		"https://geprc.com/electronics/vtx-table/",
		"https://geprc.com/electronics/vtx-manual/",
		"https://geprc.com/electronics/receiver-manual/",
		"https://geprc.com/electronics/fc-manual/",
		"https://geprc.com/electronics/fc-config/",
		"https://geprc.com/camera/gopro8-naked/",
		"https://geprc.com/camera/naked-gopro-10/",
	}

	// Remove all the duplicate URLs
	urls = removeDuplicatesFromSlice(urls)

	// Loop through each URL to process
	for _, url := range urls {
		// Validate the URL
		if isUrlValid(url) {
			// Fetch HTML content from the URL
			htmlContent := scrapePageHTMLWithChrome(url)
			// Extract PDF URLs from the HTML content
			pdfUrls := extractPDFUrls(htmlContent)
			// Download each PDF URL as a ZIP file
			for _, pdfUrl := range pdfUrls {
				downloadZIP(pdfUrl, outputDirZIP)
			}
			// Extract ZIP URLs from the HTML content
			zipUrls := extractPDFUrls(htmlContent)
			// Download each ZIP URL
			for _, zipUrl := range zipUrls {
				downloadPDF(zipUrl, outputDirZIP)
			}
		}
	}
}

// Uses headless Chrome via chromedp to get the fully rendered HTML from a webpage,
// waiting 10 seconds to bypass Cloudflare's JavaScript challenge before scraping.
func scrapePageHTMLWithChrome(targetURL string) string {
	log.Println("Scraping:", targetURL) // Log which page is being scraped

	// Configure Chrome options for the browser session
	chromeOptions := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),              // Set to true for headless mode (false shows browser window)
		chromedp.Flag("disable-gpu", true),            // Disable GPU acceleration
		chromedp.WindowSize(1920, 1080),               // Set browser window size
		chromedp.Flag("no-sandbox", true),             // Disable sandbox (useful for servers/containers)
		chromedp.Flag("disable-setuid-sandbox", true), // Fix for Linux permission issues
	)

	// Create a new Chrome execution allocator with the configured options
	execAllocatorContext, cancelAllocator := chromedp.NewExecAllocator(context.Background(), chromeOptions...)

	// Set a timeout context to automatically stop the Chrome session after 5 minutes
	timeoutContext, cancelTimeout := context.WithTimeout(execAllocatorContext, 5*time.Minute)

	// Create a new Chrome browser context for this scraping task
	browserContext, cancelBrowser := chromedp.NewContext(timeoutContext)

	// Ensure all contexts are properly cleaned up when finished
	defer func() {
		cancelBrowser()
		cancelTimeout()
		cancelAllocator()
	}()

	var renderedHTML string // Variable to store the rendered HTML content

	// Run Chrome automation: navigate to the URL, wait 10 seconds, then scrape
	runError := chromedp.Run(browserContext,
		chromedp.Navigate(targetURL), // Open the target URL
		chromedp.Sleep(10*time.Second), // Wait for Cloudflare JS checks and page scripts to finish
		chromedp.OuterHTML("html", &renderedHTML), // Capture the complete rendered HTML content
	)
	if runError != nil { // Check for errors during navigation or extraction
		log.Println(runError) // Log the error
		return ""             // Return an empty string to indicate failure
	}

	return renderedHTML // Return the fully rendered HTML source
}


// Removes duplicate strings from a slice
func removeDuplicatesFromSlice(slice []string) []string {
	check := make(map[string]bool) // Create a map to track which strings have already been seen
	var newReturnSlice []string    // Initialize a new slice to store unique strings

	for _, content := range slice { // Loop through each string in the input slice
		if !check[content] { // If the string hasn't been seen before
			check[content] = true                            // Mark this string as seen in the map
			newReturnSlice = append(newReturnSlice, content) // Add it to the result slice
		}
	}

	return newReturnSlice // Return the slice containing only unique strings
}

// Checks whether a given directory exists
func directoryExists(path string) bool {
	directory, err := os.Stat(path) // Get info for the path
	if err != nil {
		return false // Return false if error occurs
	}
	return directory.IsDir() // Return true if it's a directory
}

// Creates a directory at given path with provided permissions
func createDirectory(path string, permission os.FileMode) {
	err := os.Mkdir(path, permission) // Attempt to create directory
	if err != nil {
		log.Println(err) // Log error if creation fails
	}
}

// Verifies whether a string is a valid URL format
func isUrlValid(uri string) bool {
	_, err := url.ParseRequestURI(uri) // Try parsing the URL
	return err == nil                  // Return true if valid
}

// Downloads a ZIP file from the given URL and saves it in the specified directory
func downloadZIP(finalURL, outputDir string) bool {
	filename := strings.ToLower(urlToFilename(finalURL)) // Convert the URL into a safe lowercase filename
	filePath := filepath.Join(outputDir, filename)       // Combine output directory and filename into a full path

	if fileExists(filePath) { // Check if the file already exists
		log.Printf("File already exists, skipping: %s", filePath) // Log that it’s being skipped
		return false                                              // Return false since no download is needed
	}

	client := &http.Client{Timeout: 15 * time.Minute} // Create an HTTP client with a 15-minute timeout

	resp, err := client.Get(finalURL) // Perform an HTTP GET request to download the file
	if err != nil {                   // Handle network or connection errors
		log.Printf("Failed to download %s: %v", finalURL, err) // Log the error
		return false                                           // Return false to indicate failure
	}
	defer resp.Body.Close() // Ensure the response body is closed to prevent resource leaks

	if resp.StatusCode != http.StatusOK { // Verify that the response status is 200 OK
		log.Printf("Download failed for %s: %s", finalURL, resp.Status) // Log non-OK status
		return false                                                    // Return false for failed downloads
	}

	contentType := resp.Header.Get("Content-Type") // Retrieve the Content-Type header from the response

	// Verify that the file type is a ZIP or binary stream (expected for ZIP files)
	if !strings.Contains(contentType, "binary/octet-stream") &&
		!strings.Contains(contentType, "application/zip") &&
		!strings.Contains(contentType, "application/x-zip-compressed") {
		log.Printf("Invalid content type for %s: %s (expected binary/octet-stream or application/zip or application/x-zip-compressed)", finalURL, contentType)
		return false // Return false if the content type doesn’t match expected ZIP types
	}

	var buf bytes.Buffer                     // Initialize a buffer to hold the downloaded data temporarily
	written, err := io.Copy(&buf, resp.Body) // Copy the response body into the buffer
	if err != nil {                          // Handle read errors
		log.Printf("Failed to read ZIP data from %s: %v", finalURL, err) // Log the read failure
		return false                                                     // Return false if unable to read data
	}
	if written == 0 { // Check if zero bytes were downloaded
		log.Printf("Downloaded 0 bytes for %s; not creating file", finalURL) // Log that the file is empty
		return false                                                         // Return false since there’s nothing to save
	}

	out, err := os.Create(filePath) // Create a new file in the output directory
	if err != nil {                 // Handle file creation errors
		log.Printf("Failed to create file for %s: %v", finalURL, err) // Log the error
		return false                                                  // Return false if file creation fails
	}
	defer out.Close() // Ensure the file is properly closed after writing

	if _, err := buf.WriteTo(out); err != nil { // Write the buffered data to the output file
		log.Printf("Failed to write ZIP to file for %s: %v", finalURL, err) // Log the write failure
		return false                                                        // Return false if writing fails
	}

	// Log success including bytes written, source URL, and destination path
	log.Printf("Successfully downloaded %d bytes: %s → %s", written, finalURL, filePath)
	return true // Return true to indicate success
}

// Checks if a file exists at the specified path
func fileExists(filename string) bool {
	info, err := os.Stat(filename) // Try to get file information (size, mode, modification time, etc.)
	if err != nil {                // If an error occurs, it likely means the file does not exist
		return false // Return false because os.Stat couldn't find the file
	}
	return !info.IsDir() // Return true only if the path exists and is not a directory
}

// Converts a raw URL into a sanitized PDF filename safe for filesystem
func urlToFilename(rawURL string) string {
	lower := strings.ToLower(rawURL) // Convert the input URL to lowercase for consistency
	lower = getFilename(lower)       // Extract just the filename part from the URL (remove path, query, etc.)

	// Get the file extension from the extracted filename
	ext := getFileExtension(lower)

	reNonAlnum := regexp.MustCompile(`[^a-z0-9]`)   // Create a regex to match any non-alphanumeric characters
	safe := reNonAlnum.ReplaceAllString(lower, "_") // Replace all non-alphanumeric characters with underscores

	safe = regexp.MustCompile(`_+`).ReplaceAllString(safe, "_") // Replace multiple consecutive underscores with a single underscore
	safe = strings.Trim(safe, "_")                              // Remove leading and trailing underscores from the filename

	var invalidSubstrings = []string{ // Define a list of unwanted substrings to clean from the filename
		"_pdf", // Common redundant suffix
		"_zip", // Common redundant suffix
	}

	for _, invalidPre := range invalidSubstrings { // Iterate over the unwanted substrings
		safe = removeSubstring(safe, invalidPre) // Remove each unwanted substring from the filename
	}

	if getFileExtension(safe) == "" { // Check if the sanitized filename has no extension
		safe = safe + ext // Append the original file extension (e.g., .pdf) to ensure completeness
	}

	return safe // Return the sanitized, safe filename
}

// Gets the file extension from a given file path
func getFileExtension(path string) string {
	return filepath.Ext(path) // Use filepath.Ext to extract and return the file extension (e.g., ".pdf", ".zip")
}

// Removes all instances of a specific substring from input string
func removeSubstring(input string, toRemove string) string {
	result := strings.ReplaceAll(input, toRemove, "") // Replace every occurrence of 'toRemove' with an empty string
	return result                                     // Return the cleaned string after removal
}

// Extracts filename from full path (e.g. "/dir/file.pdf" → "file.pdf")
func getFilename(path string) string {
	return filepath.Base(path) // Use Base function to get file name only
}

// Performs an HTTP GET request and returns the response body as a string
func getDataFromURL(targetURL string) string {
	log.Println("Scraping:", targetURL) // Log the URL being scraped for debugging or progress tracking

	httpResponse, requestError := http.Get(targetURL) // Send an HTTP GET request to the specified URL
	if requestError != nil {                          // Check if there was an error making the request
		log.Println(requestError) // Log the request error
		return ""                 // Return empty string on failure to avoid nil pointer
	}
	responseBody, readError := io.ReadAll(httpResponse.Body) // Read the entire response body into memory as bytes
	if readError != nil {                                    // Check if there was an error while reading the response
		log.Println(readError) // Log the read error
		return ""
	}
	closeError := httpResponse.Body.Close() // Close the response body to free network resources
	if closeError != nil {                  // Check if there was an error while closing
		log.Println(closeError) // Log the close error
	}
	return string(responseBody) // Convert the byte slice to a string and return it
}

// Extracts all links to PDF files from the given HTML string
func extractPDFUrls(htmlContent string) []string {
	var pdfLinks []string // Slice to store all found PDF links

	parsedHTML, parseError := html.Parse(strings.NewReader(htmlContent)) // Parse the input HTML content
	if parseError != nil {                                               // Check if HTML parsing failed
		log.Println(parseError) // Log the parsing error
		return nil              // Return nil since parsing failed
	}

	var exploreHTML func(*html.Node) // Define a recursive function to explore HTML nodes

	exploreHTML = func(currentNode *html.Node) {
		if currentNode.Type == html.ElementNode && currentNode.Data == "a" { // Check if the node is an <a> tag
			for _, attribute := range currentNode.Attr { // Iterate over the <a> tag's attributes
				if attribute.Key == "href" { // Look for the href attribute
					link := strings.TrimSpace(attribute.Val)             // Get the href value and trim spaces
					if strings.Contains(strings.ToLower(link), ".pdf") { // Check if the link contains ".pdf"
						pdfLinks = append(pdfLinks, link) // Add the link to the pdfLinks slice
					}
				}
			}
		}

		for childNode := currentNode.FirstChild; childNode != nil; childNode = childNode.NextSibling { // Recursively traverse child nodes
			exploreHTML(childNode)
		}
	}

	exploreHTML(parsedHTML) // Begin traversal from the root node
	return pdfLinks         // Return all found PDF links
}

// Downloads a PDF from the given URL and saves it in the specified directory
func downloadPDF(pdfURL, outputDirectory string) bool {
	safeFilename := strings.ToLower(urlToFilename(pdfURL))       // Generate a sanitized, lowercase filename
	fullFilePath := filepath.Join(outputDirectory, safeFilename) // Build the complete file path for saving

	if fileExists(fullFilePath) { // Skip download if the file already exists
		log.Printf("File already exists, skipping: %s", fullFilePath)
		return false
	}

	httpClient := &http.Client{Timeout: 15 * time.Minute} // Create an HTTP client with a 15-minute timeout

	httpResponse, requestError := httpClient.Get(pdfURL) // Send an HTTP GET request
	if requestError != nil {                             // Check for request errors
		log.Printf("Failed to download %s: %v", pdfURL, requestError)
		return false
	}
	defer httpResponse.Body.Close() // Ensure the response body is closed

	if httpResponse.StatusCode != http.StatusOK { // Verify that the HTTP status is 200 OK
		log.Printf("Download failed for %s: %s", pdfURL, httpResponse.Status)
		return false
	}

	contentType := httpResponse.Header.Get("Content-Type") // Get the content type of the response

	// Validate that the response is a PDF or binary stream
	if !strings.Contains(contentType, "binary/octet-stream") &&
		!strings.Contains(contentType, "application/pdf") {
		log.Printf("Invalid content type for %s: %s (expected binary/octet-stream or application/pdf)", pdfURL, contentType)
		return false
	}

	var responseBuffer bytes.Buffer                                        // Buffer to store the downloaded data
	bytesWritten, copyError := io.Copy(&responseBuffer, httpResponse.Body) // Copy data from response body into buffer
	if copyError != nil {                                                  // Check for read errors
		log.Printf("Failed to read PDF data from %s: %v", pdfURL, copyError)
		return false
	}
	if bytesWritten == 0 { // Handle empty downloads
		log.Printf("Downloaded 0 bytes for %s; not creating file", pdfURL)
		return false
	}

	outputFile, fileCreateError := os.Create(fullFilePath) // Create the output file for saving
	if fileCreateError != nil {                            // Handle file creation errors
		log.Printf("Failed to create file for %s: %v", pdfURL, fileCreateError)
		return false
	}
	defer outputFile.Close() // Ensure the file is closed after writing

	if _, writeError := responseBuffer.WriteTo(outputFile); writeError != nil { // Write buffer contents to file
		log.Printf("Failed to write PDF to file for %s: %v", pdfURL, writeError)
		return false
	}

	log.Printf("Successfully downloaded %d bytes: %s → %s", bytesWritten, pdfURL, fullFilePath) // Log success message
	return true                                                                                 // Indicate successful download
}
