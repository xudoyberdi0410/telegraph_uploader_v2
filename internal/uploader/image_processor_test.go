package uploader

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

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

	processed, err := processImage(imgPath, settings)
	if err != nil {
		t.Fatalf("processImage failed: %v", err)
	}

	if processed.Size == 0 {
		t.Error("processed content is empty")
	}

	// We can't easily check dimensions of the WebP result without decoding it back, 
	// but we can check it didn't error and returned bytes.
	// Also check filename change
	if filepath.Ext(processed.FileName) != ".webp" {
		t.Errorf("expected .webp extension, got %s", filepath.Ext(processed.FileName))
	}
}

func TestProcessImage_NoResize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "imgtest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	imgPath := createTestImage(t, tmpDir, "small.png", 500, 500)

	settings := ResizeSettings{
		Resize:      true,
		ResizeTo:    1000,
		WebpQuality: 80,
	}

	processed, err := processImage(imgPath, settings)
	if err != nil {
		t.Fatalf("processImage failed: %v", err)
	}

	// Should still convert to webp
	if filepath.Ext(processed.FileName) != ".webp" {
		t.Errorf("expected .webp extension, got %s", filepath.Ext(processed.FileName))
	}
}
