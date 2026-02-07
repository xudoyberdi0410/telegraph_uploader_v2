package uploader

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/chai2010/webp"
)

// Helper to create a test image
func createTestImage(t *testing.T, dir string, name string, width, height int) string {
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill with some color
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{100, 100, 100, 255})
		}
	}

	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestProcessImage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "imgtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	imgPath := createTestImage(t, tmpDir, "test.png", 2000, 2000)

	settings := ResizeSettings{
		Resize:      true,
		ResizeTo:    1000,
		WebpQuality: 80,
	}

	data, err := os.ReadFile(imgPath)
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	processed, err := processImage(data, filepath.Base(imgPath), settings)
	if err != nil {
		t.Fatalf("processImage failed: %v", err)
	}

	if processed.Size == 0 {
		t.Error("processed content is empty")
	}

	if filepath.Ext(processed.FileName) != ".webp" {
		t.Errorf("expected .webp extension, got %s", filepath.Ext(processed.FileName))
	}

	// Verify dimensions of result
	img, err := webp.Decode(processed.Content)
	if err != nil {
		t.Fatalf("failed to decode result webp: %v", err)
	}

	if img.Bounds().Dx() > 1000 {
		t.Errorf("expected width <= 1000, got %d", img.Bounds().Dx())
	}
	// imaging.Resize maintains aspect ratio. 2000x2000 -> 1000x1000
	if img.Bounds().Dx() != 1000 {
		t.Errorf("expected width 1000, got %d", img.Bounds().Dx())
	}
}

func TestProcessImage_NoResize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "imgtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Image smaller than resize target
	imgPath := createTestImage(t, tmpDir, "small.png", 500, 500)

	settings := ResizeSettings{
		Resize:      true,
		ResizeTo:    1000,
		WebpQuality: 80,
	}

	data, err := os.ReadFile(imgPath)
	if err != nil {
		t.Fatalf("failed to read test image: %v", err)
	}

	processed, err := processImage(data, filepath.Base(imgPath), settings)
	if err != nil {
		t.Fatalf("processImage failed: %v", err)
	}

	img, err := webp.Decode(bytes.NewReader(processed.Content.Bytes()))
	if err != nil {
		t.Fatalf("failed to decode result webp: %v", err)
	}

	if img.Bounds().Dx() != 500 {
		t.Errorf("expected width 500 (no resize), got %d", img.Bounds().Dx())
	}
}

func TestProcessImage_InvalidFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "imgtest_invalid")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a text file
	txtPath := filepath.Join(tmpDir, "not_an_image.txt")
	os.WriteFile(txtPath, []byte("I am not an image"), 0644)

	data, err := os.ReadFile(txtPath)
	if err != nil {
		t.Fatal(err)
	}

	_, err = processImage(data, "not_an_image.txt", ResizeSettings{})
	if err == nil {
		t.Error("expected error for text file")
	}

	// For non-existent file, we can't read it, so processImage can't be called.
	// We should test that processImage handles nil/empty data or invalid data gracefully if we want.
	// But the original test was testing file opening failure.
	// Since processImage no longer opens file, we can't test "missing file" error from processImage.
	// But we can test empty byte slice.
	_, err = processImage([]byte{}, "missing.png", ResizeSettings{})
	if err == nil {
		t.Error("expected error for empty data")
	}
}

func TestProcessImage_CorruptedImage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "imgtest_corrupt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a file with png extension but bad content
	badPath := filepath.Join(tmpDir, "bad.png")
	os.WriteFile(badPath, []byte("\x89PNG\r\n\x1a\n...broken..."), 0644)

	data, err := os.ReadFile(badPath)
	if err != nil {
		t.Fatal(err)
	}

	_, err = processImage(data, "bad.png", ResizeSettings{})
	if err == nil {
		t.Error("expected error for corrupted image")
	}
}