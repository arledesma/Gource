package main

import (
	"fmt"

	"github.com/arledesma/gource-tui/model"
)

// Run this to verify terminal pixel size detection.
func main() {
	fmt.Println("Querying terminal pixel dimensions...")
	size := model.DetectTermPixelSize()

	fmt.Printf("Window pixels: %dx%d\n", size.PixW, size.PixH)
	fmt.Printf("Cell pixels:   %dx%d\n", size.CellW, size.CellH)

	if size.PixW > 0 && size.PixH > 0 {
		fmt.Println("Detection: OK")
	} else {
		fmt.Println("Detection: FAILED (terminal may not support CSI 14t/16t)")
	}
}
