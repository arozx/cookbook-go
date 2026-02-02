package images

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/mattn/go-sixel"
)

// ImageProtocol represents different terminal image protocols
type ImageProtocol int

const (
	ProtocolHalfBlock ImageProtocol = iota // Unicode half-block characters (works everywhere)
	ProtocolSixel                          // Sixel graphics (xterm, mlterm, foot, etc.)
	ProtocolKitty                          // Kitty graphics protocol
	ProtocolITerm2                         // iTerm2 inline images
)

// Downloader handles downloading and processing recipe images
type Downloader struct {
	client   *http.Client
	cacheDir string
	protocol ImageProtocol
}

// NewDownloader creates a new image downloader
func NewDownloader(cacheDir string) (*Downloader, error) {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}

	// Auto-detect best protocol
	protocol := detectProtocol()

	return &Downloader{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheDir: cacheDir,
		protocol: protocol,
	}, nil
}

// SetProtocol allows manually setting the image protocol
func (d *Downloader) SetProtocol(p ImageProtocol) {
	d.protocol = p
}

// GetProtocol returns the current protocol
func (d *Downloader) GetProtocol() ImageProtocol {
	return d.protocol
}

// detectProtocol tries to detect the best supported protocol
func detectProtocol() ImageProtocol {
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")
	kitty := os.Getenv("KITTY_WINDOW_ID")

	// Check for Kitty
	if kitty != "" {
		return ProtocolKitty
	}

	// Check for iTerm2
	if termProgram == "iTerm.app" {
		return ProtocolITerm2
	}

	// Check for Sixel support (common in xterm, mlterm, foot, etc.)
	// These terminals commonly support sixel
	sixelTerms := []string{"xterm", "mlterm", "foot", "yaft", "mintty", "msys"}
	for _, t := range sixelTerms {
		if strings.Contains(term, t) {
			return ProtocolSixel
		}
	}

	// Check SIXEL env var that some terminals set
	if os.Getenv("SIXEL") == "1" {
		return ProtocolSixel
	}

	// Default to half-block (works everywhere)
	return ProtocolHalfBlock
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

// RenderImage renders an image using the best available protocol
func (d *Downloader) RenderImage(imagePath string, width, height int) (string, error) {
	switch d.protocol {
	case ProtocolKitty:
		return ImageToKitty(imagePath, width, height)
	case ProtocolITerm2:
		return ImageToITerm2(imagePath, width, height)
	case ProtocolSixel:
		return ImageToSixel(imagePath, width, height)
	default:
		return ImageToHalfBlock(imagePath, width, height)
	}
}

// RenderImageSafe renders an image using half-block characters only.
// This is safe for use inside scrolling viewports and TUI components
// because it uses regular text characters rather than terminal graphics protocols.
func (d *Downloader) RenderImageSafe(imagePath string, width, height int) (string, error) {
	return ImageToHalfBlock(imagePath, width, height)
}

// ImageToSixel converts an image to Sixel format
func ImageToSixel(imagePath string, width, height int) (string, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("opening image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("decoding image: %w", err)
	}

	// Resize maintaining aspect ratio
	// Sixel uses 6 pixels per character row
	pixelHeight := height * 6
	resized := imaging.Fit(img, width*2, pixelHeight, imaging.Lanczos)

	var buf bytes.Buffer
	enc := sixel.NewEncoder(&buf)
	enc.Dither = true

	if err := enc.Encode(resized); err != nil {
		return "", fmt.Errorf("encoding sixel: %w", err)
	}

	return buf.String(), nil
}

// ImageToKitty converts an image to Kitty graphics protocol format
func ImageToKitty(imagePath string, width, height int) (string, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("opening image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("decoding image: %w", err)
	}

	// Resize to fit terminal cells (each cell is roughly 10x20 pixels)
	pixelWidth := width * 10
	pixelHeight := height * 20
	resized := imaging.Fit(img, pixelWidth, pixelHeight, imaging.Lanczos)

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, resized); err != nil {
		return "", fmt.Errorf("encoding png: %w", err)
	}

	// Base64 encode
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Kitty protocol: split into chunks of 4096 bytes
	var result strings.Builder
	chunkSize := 4096

	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		chunk := encoded[i:end]

		// m=1 means more chunks follow, m=0 means last chunk
		more := 1
		if end >= len(encoded) {
			more = 0
		}

		if i == 0 {
			// First chunk: include format and action
			// a=T: transmit and display
			// f=100: PNG format
			// c/r: columns/rows (optional, let terminal auto-size)
			result.WriteString(fmt.Sprintf("\x1b_Ga=T,f=100,m=%d;%s\x1b\\", more, chunk))
		} else {
			// Subsequent chunks
			result.WriteString(fmt.Sprintf("\x1b_Gm=%d;%s\x1b\\", more, chunk))
		}
	}

	result.WriteString("\n")
	return result.String(), nil
}

// ImageToITerm2 converts an image to iTerm2 inline image format
func ImageToITerm2(imagePath string, width, height int) (string, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("opening image: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("decoding image: %w", err)
	}

	// Resize to fit
	pixelWidth := width * 10
	pixelHeight := height * 20
	resized := imaging.Fit(img, pixelWidth, pixelHeight, imaging.Lanczos)

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, resized); err != nil {
		return "", fmt.Errorf("encoding png: %w", err)
	}

	// Base64 encode
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	// iTerm2 inline image protocol
	// OSC 1337 ; File=[args] : base64 ST
	// width/height in cells, preserveAspectRatio=1
	result := fmt.Sprintf("\x1b]1337;File=inline=1;width=%d;height=%d;preserveAspectRatio=1:%s\x07\n",
		width, height, encoded)

	return result, nil
}

// ImageToHalfBlock uses half-block characters for higher resolution (fallback)
func ImageToHalfBlock(imagePath string, width, height int) (string, error) {
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

// ImageToASCII converts an image to ASCII art for terminal display
func ImageToASCII(imagePath string, width, height int) (string, error) {
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

// GetCacheDir returns the cache directory path
func (d *Downloader) GetCacheDir() string {
	return d.cacheDir
}

// ClearGraphics returns escape sequences to clear any rendered graphics
// This should be called when switching away from image display
func (d *Downloader) ClearGraphics() string {
	switch d.protocol {
	case ProtocolKitty:
		// Kitty: delete all images
		// a=d: action=delete, d=A: delete all images
		return "\x1b_Ga=d,d=A\x1b\\"
	case ProtocolITerm2:
		// iTerm2 doesn't have a specific clear command
		// We rely on the screen being redrawn
		return ""
	case ProtocolSixel:
		// Sixel doesn't have a clear command either
		// The terminal should overwrite on redraw
		return ""
	default:
		return ""
	}
}

// ProtocolName returns a human-readable name for the protocol
func (p ImageProtocol) String() string {
	switch p {
	case ProtocolSixel:
		return "Sixel"
	case ProtocolKitty:
		return "Kitty"
	case ProtocolITerm2:
		return "iTerm2"
	default:
		return "Half-Block"
	}
}
