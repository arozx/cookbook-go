package images

import (
	"crypto/md5"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
)

// Downloader handles downloading and processing recipe images
type Downloader struct {
	client   *http.Client
	cacheDir string
}

// NewDownloader creates a new image downloader
func NewDownloader(cacheDir string) (*Downloader, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}

	return &Downloader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheDir: cacheDir,
	}, nil
}

// Download fetches an image from URL and saves it locally
func (d *Downloader) Download(imageURL string) (string, error) {
	// Generate cache filename
	hash := md5.Sum([]byte(imageURL))
	ext := filepath.Ext(imageURL)
	if ext == "" || len(ext) > 5 {
		ext = ".jpg"
	}
	// Remove query params from extension
	if idx := strings.Index(ext, "?"); idx != -1 {
		ext = ext[:idx]
	}
	filename := fmt.Sprintf("%x%s", hash, ext)
	localPath := filepath.Join(d.cacheDir, filename)

	// Check if already cached
	if _, err := os.Stat(localPath); err == nil {
		return localPath, nil
	}

	// Download the image
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := d.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	// Create the file
	file, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("creating file: %w", err)
	}
	defer file.Close()

	// Copy the content
	if _, err := io.Copy(file, resp.Body); err != nil {
		os.Remove(localPath)
		return "", fmt.Errorf("saving image: %w", err)
	}

	return localPath, nil
}

// DownloadAll downloads all images for a recipe
func (d *Downloader) DownloadAll(imageURLs []string) []string {
	var paths []string
	for _, url := range imageURLs {
		if path, err := d.Download(url); err == nil {
			paths = append(paths, path)
		}
	}
	return paths
}

// ImageToASCII converts an image to ASCII art for terminal display
func ImageToASCII(imagePath string, width, height int) (string, error) {
	// Open the image
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("opening image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("decoding image: %w", err)
	}

	// Resize the image
	resized := imaging.Resize(img, width, height, imaging.Lanczos)

	// ASCII characters from dark to light
	ascii := []rune(" .:-=+*#%@")

	var sb strings.Builder
	bounds := resized.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := resized.At(x, y).RGBA()
			// Convert to grayscale
			gray := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)) / 256
			// Map to ASCII character
			idx := int(gray / 256 * float64(len(ascii)-1))
			if idx >= len(ascii) {
				idx = len(ascii) - 1
			}
			sb.WriteRune(ascii[idx])
		}
		sb.WriteRune('\n')
	}

	return sb.String(), nil
}

// ImageToANSI converts an image to colored ANSI blocks for terminal display
func ImageToANSI(imagePath string, width, height int) (string, error) {
	// Open the image
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("opening image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("decoding image: %w", err)
	}

	// Resize the image (height/2 because terminal characters are ~2:1 aspect ratio)
	resized := imaging.Resize(img, width, height/2, imaging.Lanczos)

	var sb strings.Builder
	bounds := resized.Bounds()

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := resized.At(x, y).RGBA()
			// Use true color ANSI escape codes
			sb.WriteString(fmt.Sprintf("\x1b[48;2;%d;%d;%dm ", r>>8, g>>8, b>>8))
		}
		sb.WriteString("\x1b[0m\n")
	}

	return sb.String(), nil
}

// ImageToHalfBlock uses half-block characters for higher resolution
func ImageToHalfBlock(imagePath string, width, height int) (string, error) {
	// Open the image
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("opening image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("decoding image: %w", err)
	}

	// Resize - we'll use 2 pixels vertically per character
	resized := imaging.Resize(img, width, height, imaging.Lanczos)

	var sb strings.Builder
	bounds := resized.Bounds()

	// Process 2 rows at a time
	for y := bounds.Min.Y; y < bounds.Max.Y-1; y += 2 {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Top pixel
			r1, g1, b1, _ := resized.At(x, y).RGBA()
			// Bottom pixel
			r2, g2, b2, _ := resized.At(x, y+1).RGBA()

			// Use upper half block with foreground color for top, background for bottom
			sb.WriteString(fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀",
				r1>>8, g1>>8, b1>>8,
				r2>>8, g2>>8, b2>>8))
		}
		sb.WriteString("\x1b[0m\n")
	}

	return sb.String(), nil
}

// GetCacheDir returns the cache directory path
func (d *Downloader) GetCacheDir() string {
	return d.cacheDir
}
