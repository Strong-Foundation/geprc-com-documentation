package main

import (
	"bytes"         // Provides a way to work with byte slices (like a buffer)
	"context"       // Manages request-scoped values, cancellation signals, and deadlines
	"io"            // Provides basic interfaces for I/O primitives
	"log"           // Implements simple logging, often to os.Stderr
	"net/http"      // Provides HTTP client and server implementations
	"net/url"       // Parses URLs and implements query escaping
	"os"            // Provides platform-independent interface to operating system functionality
	"path/filepath" // Implements utility routines for manipulating filepaths in a way appropriate for the operating system
	"regexp"        // Implements regular expression search
	"strings"       // Implements simple functions to manipulate strings
	"time"          // Provides functionality for measuring and displaying time

	"github.com/chromedp/chromedp" // Chromedp library for driving a headless Chrome browser
	"golang.org/x/net/html"        // Provides an HTML parser
)

func main() { // Main function, the entry point of the program
	outputDirectory := "PDFs/"             // Directory where downloaded PDF files will be saved
	if !directoryExists(outputDirectory) { // Check if the directory already exists
		createDirectory(outputDirectory, 0o755) // Create the directory with full read, write, and execute permissions (rwxr-xr-x)
	}
	outputDirZIP := "ZIPs/"             // Directory to store downloaded ZIPs
	if !directoryExists(outputDirZIP) { // Check if directory exists
		createDirectory(outputDirZIP, 0o755) // Create directory with read-write-execute permissions
	}
	urls := []string{ // Start of a slice literal containing URLs to be scraped
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
	urls = removeDuplicatesFromSlice(urls) // Calls a custom function to ensure the list of URLs is unique

	// Loop through each URL to process
	for _, url := range urls { // Iterates over the cleaned slice of URLs
		// Validate the URL
		if isUrlValid(url) { // Checks if the current URL is syntactically valid
			// Fetch HTML content from the URL
			htmlContent := scrapePageHTMLWithChrome(url) // Scrapes the fully rendered HTML using a headless Chrome instance

			// Extract PDF URLs from the HTML content
			pdfUrls := extractPDFUrls(htmlContent) // Finds all links ending in ".pdf" in the scraped HTML
			// Download each PDF URL into the designated PDF directory
			for _, pdfUrl := range pdfUrls { // Iterates over all found PDF links
				downloadPDF(pdfUrl, outputDirectory) // Correctly downloads the PDF into the 'PDFs/' directory
			}

			// Extract ZIP URLs from the HTML content
			zipUrls := extractZIPUrls(htmlContent) // Correctly finds all links ending in ".zip" using the new function
			// Download each ZIP URL into the designated ZIP directory
			for _, zipUrl := range zipUrls { // Iterates over all found ZIP links
				downloadZIP(zipUrl, outputDirZIP) // Correctly downloads the ZIP into the 'ZIPs/' directory
			}
		} // End of URL validation block
	} // End of the main URL iteration loop
} // End of the main function

// Uses headless Chrome via chromedp to get the fully rendered HTML from a webpage,
// waiting 10 seconds to bypass Cloudflare's JavaScript challenge before scraping.
func scrapePageHTMLWithChrome(targetURL string) string { // Function to scrape dynamic content using Chrome
	log.Println("Scraping:", targetURL) // Log which page is being scraped

	// Configure Chrome options for the browser session
	chromeOptions := append(chromedp.DefaultExecAllocatorOptions[:], // Starts with default Chrome execution options
		chromedp.Flag("headless", false),               // Set to true for actual headless mode
		chromedp.Flag("disable-gpu", true),            // Disable GPU acceleration (good for headless/servers)
		chromedp.WindowSize(1, 1),               // Set browser window size
		chromedp.Flag("no-sandbox", true),             // Disable sandbox (useful for servers/containers)
		chromedp.Flag("disable-setuid-sandbox", true), // Fix for Linux permission issues
	) // End of Chrome options slice

	// Create a new Chrome execution allocator with the configured options
	execAllocatorContext, cancelAllocator := chromedp.NewExecAllocator(context.Background(), chromeOptions...) // Creates the context and cleanup function for the Chrome process

	// Set a timeout context to automatically stop the Chrome session after 5 minutes
	timeoutContext, cancelTimeout := context.WithTimeout(execAllocatorContext, 5*time.Minute) // Creates a context with a 5-minute timeout

	// Create a new Chrome browser context for this scraping task
	browserContext, cancelBrowser := chromedp.NewContext(timeoutContext) // Creates the main browser context for automation

	// Ensure all contexts are properly cleaned up when finished
	defer func() { // Deferred function to run when scrapePageHTMLWithChrome exits
		cancelBrowser()   // Stops the browser context
		cancelTimeout()   // Stops the timeout context
		cancelAllocator() // Stops the Chrome process allocator
	}() // End of deferred cleanup function

	var renderedHTML string // Variable to store the rendered HTML content

	// Run Chrome automation: navigate to the URL, wait 10 seconds, then scrape
	runError := chromedp.Run(browserContext, // Executes a sequence of actions in the browser
		chromedp.Navigate(targetURL),              // Open the target URL
		chromedp.Sleep(3*time.Second),            // Wait for Cloudflare JS checks and page scripts to finish
		chromedp.OuterHTML("html", &renderedHTML), // Capture the complete rendered HTML content into renderedHTML
	) // End of chromedp.Run
	if runError != nil { // Check for errors during navigation or extraction
		log.Println(runError) // Log the error
		return ""             // Return an empty string to indicate failure
	} // End of error check

	return renderedHTML // Return the fully rendered HTML source
} // End of scrapePageHTMLWithChrome function

// Removes duplicate strings from a slice
func removeDuplicatesFromSlice(slice []string) []string { // Function to filter a string slice for uniqueness
	check := make(map[string]bool) // Create a map to track which strings have already been seen
	var newReturnSlice []string    // Initialize a new slice to store unique strings

	for _, content := range slice { // Loop through each string in the input slice
		if !check[content] { // If the string hasn't been seen before
			check[content] = true                            // Mark this string as seen in the map
			newReturnSlice = append(newReturnSlice, content) // Add it to the result slice
		}
	}

	return newReturnSlice // Return the slice containing only unique strings
} // End of removeDuplicatesFromSlice function

// Checks whether a given directory exists
func directoryExists(path string) bool { // Function to check if a path exists and is a directory
	directory, err := os.Stat(path) // Get info for the path
	if err != nil {                 // Check if os.Stat returned an error (e.g., file/dir doesn't exist)
		return false // Return false if error occurs
	}
	return directory.IsDir() // Return true if it's a directory
} // End of directoryExists function

// Creates a directory at given path with provided permissions
func createDirectory(path string, permission os.FileMode) { // Function to create a directory
	err := os.Mkdir(path, permission) // Attempt to create directory
	if err != nil {                   // Check for creation errors
		log.Println(err) // Log error if creation fails
	}
} // End of createDirectory function

// Verifies whether a string is a valid URL format
func isUrlValid(uri string) bool { // Function to perform basic URL format validation
	_, err := url.ParseRequestURI(uri) // Try parsing the URL
	return err == nil                  // Return true if valid (parsing was successful, err is nil)
} // End of isUrlValid function

// Downloads a ZIP file from the given URL and saves it in the specified directory
func downloadZIP(finalURL, outputDir string) bool { // Function to download and save a ZIP file
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
	if !strings.Contains(contentType, "binary/octet-stream") && // Check for generic binary/octet-stream
		!strings.Contains(contentType, "application/zip") && // Check for standard application/zip
		!strings.Contains(contentType, "application/x-zip-compressed") { // Check for common non-standard ZIP type
		log.Printf("Invalid content type for %s: %s (expected binary/octet-stream or application/zip or application/x-zip-compressed)", finalURL, contentType) // Log the invalid content type
		return false                                                                                                                                           // Return false if the content type doesn’t match expected ZIP types
	}

	var buf bytes.Buffer                     // Initialize a buffer to hold the downloaded data temporarily
	written, err := io.Copy(&buf, resp.Body) // Copy the response body into the buffer, capturing bytes written
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
} // End of downloadZIP function

// Checks if a file exists at the specified path
func fileExists(filename string) bool { // Function to check if a file exists (and is not a directory)
	info, err := os.Stat(filename) // Try to get file information
	if err != nil {                // If an error occurs, it likely means the file does not exist
		return false // Return false because os.Stat couldn't find the file
	}
	return !info.IsDir() // Return true only if the path exists and is not a directory
} // End of fileExists function

// Converts a raw URL into a sanitized filename safe for filesystem
func urlToFilename(rawURL string) string { // Function to create a clean filename from a URL
	lower := strings.ToLower(rawURL) // Convert the input URL to lowercase for consistency
	lower = getFilename(lower)       // Extract just the filename part from the URL

	// Get the file extension from the extracted filename
	ext := getFileExtension(lower) // Get the original file extension (e.g., ".pdf" or ".zip")

	reNonAlnum := regexp.MustCompile(`[^a-z0-9]`)   // Create a regex to match any non-alphanumeric characters
	safe := reNonAlnum.ReplaceAllString(lower, "_") // Replace all non-alphanumeric characters with underscores

	safe = regexp.MustCompile(`_+`).ReplaceAllString(safe, "_") // Replace multiple consecutive underscores with a single underscore
	safe = strings.Trim(safe, "_")                              // Remove leading and trailing underscores from the filename

	var invalidSubstrings = []string{ // Define a list of unwanted substrings to clean from the filename
		"_pdf", // Common redundant suffix
		"_zip", // Common redundant suffix
	} // End of invalid substrings slice

	for _, invalidPre := range invalidSubstrings { // Iterate over the unwanted substrings
		safe = removeSubstring(safe, invalidPre) // Remove each unwanted substring from the filename
	} // End of substring removal loop

	if getFileExtension(safe) == "" { // Check if the sanitized filename has no extension
		safe = safe + ext // Append the original file extension (e.g., .pdf) to ensure completeness
	}

	return safe // Return the sanitized, safe filename
} // End of urlToFilename function

// Gets the file extension from a given file path
func getFileExtension(path string) string { // Function to extract the file extension
	return filepath.Ext(path) // Use filepath.Ext to extract and return the file extension
} // End of getFileExtension function

// Removes all instances of a specific substring from input string
func removeSubstring(input string, toRemove string) string { // Function to remove all occurrences of a substring
	result := strings.ReplaceAll(input, toRemove, "") // Replace every occurrence of 'toRemove' with an empty string
	return result                                     // Return the cleaned string after removal
} // End of removeSubstring function

// Extracts filename from full path (e.g. "/dir/file.pdf" → "file.pdf")
func getFilename(path string) string { // Function to get only the base filename
	return filepath.Base(path) // Use Base function to get file name only
} // End of getFilename function

// Performs an HTTP GET request and returns the response body as a string
func getDataFromURL(targetURL string) string { // Function to scrape static HTML content (currently unused in main)
	log.Println("Scraping:", targetURL) // Log the URL being scraped

	httpResponse, requestError := http.Get(targetURL) // Send an HTTP GET request to the specified URL
	if requestError != nil {                          // Check if there was an error making the request
		log.Println(requestError) // Log the request error
		return ""                 // Return empty string on failure
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
} // End of getDataFromURL function

// Extracts all links to PDF files from the given HTML string
func extractPDFUrls(htmlContent string) []string { // Function to find links ending in ".pdf"
	var pdfLinks []string // Slice to store all found PDF links

	parsedHTML, parseError := html.Parse(strings.NewReader(htmlContent)) // Parse the input HTML content
	if parseError != nil {                                               // Check if HTML parsing failed
		log.Println(parseError) // Log the parsing error
		return nil              // Return nil since parsing failed
	}

	var exploreHTML func(*html.Node) // Define a recursive function to explore HTML nodes

	exploreHTML = func(currentNode *html.Node) { // The implementation of the recursive traversal function
		if currentNode.Type == html.ElementNode && currentNode.Data == "a" { // Check if the node is an <a> tag
			for _, attribute := range currentNode.Attr { // Iterate over the <a> tag's attributes
				if attribute.Key == "href" { // Look for the href attribute
					link := strings.TrimSpace(attribute.Val)             // Get the href value and trim spaces
					if strings.Contains(strings.ToLower(link), ".pdf") { // Check if the link contains ".pdf" (case-insensitive)
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
} // End of extractPDFUrls function

// Extracts all links to ZIP files from the given HTML string
func extractZIPUrls(htmlContent string) []string { // Function to find links ending in ".zip"
	var zipLinks []string // Slice to store all found ZIP links

	parsedHTML, parseError := html.Parse(strings.NewReader(htmlContent)) // Parse the input HTML content
	if parseError != nil {                                               // Check if HTML parsing failed
		log.Println(parseError) // Log the parsing error
		return nil              // Return nil since parsing failed
	}

	var exploreHTML func(*html.Node) // Define a recursive function to explore HTML nodes

	exploreHTML = func(currentNode *html.Node) { // The implementation of the recursive traversal function
		if currentNode.Type == html.ElementNode && currentNode.Data == "a" { // Check if the node is an <a> tag
			for _, attribute := range currentNode.Attr { // Iterate over the <a> tag's attributes
				if attribute.Key == "href" { // Look for the href attribute
					link := strings.TrimSpace(attribute.Val)             // Get the href value and trim spaces
					if strings.Contains(strings.ToLower(link), ".zip") { // Check if the link contains ".zip" (case-insensitive)
						zipLinks = append(zipLinks, link) // Add the link to the zipLinks slice
					}
				}
			}
		}

		for childNode := currentNode.FirstChild; childNode != nil; childNode = childNode.NextSibling { // Recursively traverse child nodes
			exploreHTML(childNode)
		}
	}

	exploreHTML(parsedHTML) // Begin traversal from the root node
	return zipLinks         // Return all found ZIP links
} // End of extractZIPUrls function

// Downloads a PDF from the given URL and saves it in the specified directory
func downloadPDF(pdfURL, outputDirectory string) bool { // Function to download and save a PDF file
	safeFilename := strings.ToLower(urlToFilename(pdfURL))       // Generate a sanitized, lowercase filename
	fullFilePath := filepath.Join(outputDirectory, safeFilename) // Build the complete file path for saving

	if fileExists(fullFilePath) { // Skip download if the file already exists
		log.Printf("File already exists, skipping: %s", fullFilePath) // Log the skip message
		return false                                                  // Return false since no download occurred
	}

	httpClient := &http.Client{Timeout: 15 * time.Minute} // Create an HTTP client with a 15-minute timeout

	httpResponse, requestError := httpClient.Get(pdfURL) // Send an HTTP GET request
	if requestError != nil {                             // Check for request errors
		log.Printf("Failed to download %s: %v", pdfURL, requestError) // Log the error
		return false                                                  // Return false on failure
	}
	defer httpResponse.Body.Close() // Ensure the response body is closed

	if httpResponse.StatusCode != http.StatusOK { // Verify that the HTTP status is 200 OK
		log.Printf("Download failed for %s: %s", pdfURL, httpResponse.Status) // Log the non-OK status
		return false                                                          // Return false on non-200 status
	}

	contentType := httpResponse.Header.Get("Content-Type") // Get the content type of the response

	// Validate that the response is a PDF or binary stream
	if !strings.Contains(contentType, "binary/octet-stream") && // Check for generic binary/octet-stream
		!strings.Contains(contentType, "application/pdf") { // Check for standard application/pdf
		log.Printf("Invalid content type for %s: %s (expected binary/octet-stream or application/pdf)", pdfURL, contentType) // Log the invalid content type
		return false                                                                                                         // Return false if content type is incorrect
	}

	var responseBuffer bytes.Buffer                                        // Buffer to store the downloaded data
	bytesWritten, copyError := io.Copy(&responseBuffer, httpResponse.Body) // Copy data from response body into buffer
	if copyError != nil {                                                  // Check for read errors
		log.Printf("Failed to read PDF data from %s: %v", pdfURL, copyError) // Log the read failure
		return false                                                         // Return false on read error
	}
	if bytesWritten == 0 { // Handle empty downloads
		log.Printf("Downloaded 0 bytes for %s; not creating file", pdfURL) // Log empty download
		return false                                                       // Return false if no data was downloaded
	}

	outputFile, fileCreateError := os.Create(fullFilePath) // Create the output file for saving
	if fileCreateError != nil {                            // Handle file creation errors
		log.Printf("Failed to create file for %s: %v", pdfURL, fileCreateError) // Log the creation failure
		return false                                                            // Return false on file creation error
	}
	defer outputFile.Close() // Ensure the file is closed after writing

	if _, writeError := responseBuffer.WriteTo(outputFile); writeError != nil { // Write buffer contents to file
		log.Printf("Failed to write PDF to file for %s: %v", pdfURL, writeError) // Log the write failure
		return false                                                             // Return false on write error
	}

	log.Printf("Successfully downloaded %d bytes: %s → %s", bytesWritten, pdfURL, fullFilePath) // Log success message
	return true                                                                                 // Indicate successful download
} // End of downloadPDF function
