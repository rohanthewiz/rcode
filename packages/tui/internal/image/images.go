package image

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/disintegration/imaging"
	"github.com/lucasb-eyer/go-colorful"
	_ "golang.org/x/image/webp"
)

func ValidateFileSize(filePath string, sizeLimit int64) (bool, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, fmt.Errorf("error getting file info: %w", err)
	}

	if fileInfo.Size() > sizeLimit {
		return true, nil
	}

	return false, nil
}

func ToString(width int, img image.Image) string {
	img = imaging.Resize(img, width, 0, imaging.Lanczos)
	b := img.Bounds()
	imageWidth := b.Max.X
	h := b.Max.Y
	str := strings.Builder{}

	for heightCounter := 0; heightCounter < h; heightCounter += 2 {
		for x := range imageWidth {
			c1, _ := colorful.MakeColor(img.At(x, heightCounter))
			color1 := lipgloss.Color(c1.Hex())

			var color2 color.Color
			if heightCounter+1 < h {
				c2, _ := colorful.MakeColor(img.At(x, heightCounter+1))
				color2 = lipgloss.Color(c2.Hex())
			} else {
				color2 = color1
			}

			str.WriteString(lipgloss.NewStyle().Foreground(color1).
				Background(color2).Render("▀"))
		}

		str.WriteString("\n")
	}

	return str.String()
}

func ImagePreview(width int, filename string) (string, error) {
	imageContent, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer imageContent.Close()

	img, _, err := image.Decode(imageContent)
	if err != nil {
		return "", err
	}

	imageString := ToString(width, img)

	return imageString, nil
}

func ImageToBytes(image image.Image) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := png.Encode(buf, image)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
