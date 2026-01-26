package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

func main() {
	fmt.Println("=== Terminal Color Detection Diagnostic ===")
	fmt.Println()

	// Environment variables
	printEnvVars()

	// Color profile detection
	printColorProfile()

	// Background color detection
	printBackgroundDetection()

	// Raw terminal query test
	printRawTerminalQuery()
}

func printEnvVars() {
	fmt.Println("--- Environment Variables ---")

	vars := []string{
		"TERM",
		"COLORTERM",
		"COLORFGBG",
		"NO_COLOR",
		"CLICOLOR",
		"CLICOLOR_FORCE",
		"TERM_PROGRAM",
		"TERM_PROGRAM_VERSION",
		"TERMINAL_EMULATOR",
		"WT_SESSION",       // Windows Terminal
		"KONSOLE_VERSION",  // KDE Konsole
		"GNOME_TERMINAL_SCREEN",
		"VTE_VERSION",
		"ITERM_SESSION_ID", // iTerm2
		"KITTY_WINDOW_ID",  // Kitty
		"ALACRITTY_LOG",    // Alacritty (older)
		"ALACRITTY_SOCKET", // Alacritty (newer)
		"WEZTERM_PANE",     // WezTerm
		"TMUX",
		"STY", // GNU Screen
	}

	for _, v := range vars {
		val := os.Getenv(v)
		if val != "" {
			fmt.Printf("  %s=%s\n", v, val)
		} else {
			fmt.Printf("  %s=(not set)\n", v)
		}
	}
	fmt.Println()
}

func printColorProfile() {
	fmt.Println("--- Color Profile Detection ---")

	// Using charmbracelet/colorprofile
	profile := colorprofile.Detect(os.Stdout, os.Environ())
	profileName := "Unknown"
	switch profile {
	case colorprofile.NoTTY:
		profileName = "NoTTY (no terminal)"
	case colorprofile.Ascii:
		profileName = "Ascii (no colors)"
	case colorprofile.ANSI:
		profileName = "ANSI (16 colors, 4-bit)"
	case colorprofile.ANSI256:
		profileName = "ANSI256 (256 colors, 8-bit)"
	case colorprofile.TrueColor:
		profileName = "TrueColor (16M colors, 24-bit)"
	}
	fmt.Printf("  colorprofile.Detect: %s\n", profileName)

	// Using muesli/termenv
	termenvOutput := termenv.NewOutput(os.Stdout)
	termenvProfile := termenvOutput.Profile
	termenvName := "Unknown"
	switch termenvProfile {
	case termenv.Ascii:
		termenvName = "Ascii"
	case termenv.ANSI:
		termenvName = "ANSI (16 colors)"
	case termenv.ANSI256:
		termenvName = "ANSI256 (256 colors)"
	case termenv.TrueColor:
		termenvName = "TrueColor (16M colors)"
	}
	fmt.Printf("  termenv.Profile: %s\n", termenvName)

	fmt.Println()
}

func printBackgroundDetection() {
	fmt.Println("--- Background Color Detection ---")

	// Lipgloss v1 uses a renderer
	renderer := lipgloss.DefaultRenderer()
	hasDarkBG := renderer.HasDarkBackground()
	fmt.Printf("  lipgloss.DefaultRenderer().HasDarkBackground: %v\n", hasDarkBG)

	// Create a fresh renderer to stdout to test if caching is the issue
	freshRenderer := lipgloss.NewRenderer(os.Stdout)
	freshHasDark := freshRenderer.HasDarkBackground()
	fmt.Printf("  lipgloss.NewRenderer(stdout).HasDarkBackground: %v\n", freshHasDark)

	// Check if they differ (indicates caching issue)
	if hasDarkBG != freshHasDark {
		fmt.Println("    ⚠ WARNING: DefaultRenderer and NewRenderer disagree!")
	}

	// Termenv background detection (more detailed)
	// Test with fresh output (no cache)
	termenvOutput := termenv.NewOutput(os.Stdout, termenv.WithColorCache(false))
	termenvHasDark := termenvOutput.HasDarkBackground()
	fmt.Printf("  termenv.NewOutput(no cache).HasDarkBackground: %v\n", termenvHasDark)

	// Test with default (may be cached)
	defaultTermenvHasDark := termenv.HasDarkBackground()
	fmt.Printf("  termenv.HasDarkBackground() [global]: %v\n", defaultTermenvHasDark)

	if termenvHasDark != defaultTermenvHasDark {
		fmt.Println("    ⚠ WARNING: Fresh vs cached termenv disagree!")
	}

	termenvBG := termenvOutput.BackgroundColor()
	if termenvBG != nil {
		rgb := termenv.ConvertToRGB(termenvBG)
		r8, g8, b8 := uint8(rgb.R*255), uint8(rgb.G*255), uint8(rgb.B*255)
		fmt.Printf("  termenv.BackgroundColor: R=%d G=%d B=%d (hex: %s)\n",
			r8, g8, b8, rgb.Hex())

		// Calculate HSL for analysis
		h, s, l := rgb.Hsl()
		fmt.Printf("    HSL: H=%.1f° S=%.2f L=%.2f\n", h, s, l)

		// WCAG luminance
		luminance := calculateLuminance(rgb.R, rgb.G, rgb.B)
		fmt.Printf("    WCAG Luminance: %.4f (dark < 0.5 < light)\n", luminance)

		// Verdict based on different methods
		fmt.Println("    --- Detection Method Comparison ---")
		fmt.Printf("    By HSL lightness (L < 0.5):    %s\n", darkOrLight(l < 0.5))
		fmt.Printf("    By WCAG luminance (L < 0.5):   %s\n", darkOrLight(luminance < 0.5))
		fmt.Printf("    By simple threshold (R+G+B)/3: %s\n", darkOrLight((rgb.R+rgb.G+rgb.B)/3 < 0.5))

		// Test with go-colorful
		cf := colorful.Color{R: rgb.R, G: rgb.G, B: rgb.B}
		_, _, cfL := cf.Hsl()
		fmt.Printf("    go-colorful HSL L:             %.4f\n", cfL)
	} else {
		fmt.Println("  termenv.BackgroundColor: nil (query failed or unsupported)")
	}

	// Termenv foreground for comparison
	termenvFG := termenvOutput.ForegroundColor()
	if termenvFG != nil {
		rgb := termenv.ConvertToRGB(termenvFG)
		r8, g8, b8 := uint8(rgb.R*255), uint8(rgb.G*255), uint8(rgb.B*255)
		fmt.Printf("  termenv.ForegroundColor: R=%d G=%d B=%d (hex: %s)\n",
			r8, g8, b8, rgb.Hex())
		h, s, l := rgb.Hsl()
		fmt.Printf("    HSL: H=%.1f° S=%.2f L=%.2f\n", h, s, l)
	} else {
		fmt.Println("  termenv.ForegroundColor: nil (query failed)")
	}

	// Test query order - does querying FG first vs BG first matter?
	fmt.Println("  --- Query Order Test ---")
	// Fresh output, query FG first
	freshOutput1 := termenv.NewOutput(os.Stdout, termenv.WithColorCache(false))
	fg1 := freshOutput1.ForegroundColor()
	bg1 := freshOutput1.BackgroundColor()
	if fg1 != nil && bg1 != nil {
		fgRGB := termenv.ConvertToRGB(fg1)
		bgRGB := termenv.ConvertToRGB(bg1)
		fmt.Printf("  Query FG then BG: FG=%s BG=%s\n", fgRGB.Hex(), bgRGB.Hex())
	}

	// Fresh output, query BG first
	freshOutput2 := termenv.NewOutput(os.Stdout, termenv.WithColorCache(false))
	bg2 := freshOutput2.BackgroundColor()
	fg2 := freshOutput2.ForegroundColor()
	if fg2 != nil && bg2 != nil {
		fgRGB := termenv.ConvertToRGB(fg2)
		bgRGB := termenv.ConvertToRGB(bg2)
		fmt.Printf("  Query BG then FG: FG=%s BG=%s\n", fgRGB.Hex(), bgRGB.Hex())
	}

	// Test with delay between queries
	fmt.Println("  --- With 100ms delay between queries ---")
	freshOutput3 := termenv.NewOutput(os.Stdout, termenv.WithColorCache(false))
	bg3 := freshOutput3.BackgroundColor()
	time.Sleep(100 * time.Millisecond)
	fg3 := freshOutput3.ForegroundColor()
	if fg3 != nil && bg3 != nil {
		fgRGB := termenv.ConvertToRGB(fg3)
		bgRGB := termenv.ConvertToRGB(bg3)
		fmt.Printf("  With delay: FG=%s BG=%s\n", fgRGB.Hex(), bgRGB.Hex())
		if fgRGB.Hex() != bgRGB.Hex() {
			fmt.Println("    ✓ Delay fixed the issue!")
		}
	}

	// Compare FG vs BG lightness (alternative detection method)
	if termenvBG != nil && termenvFG != nil {
		bgRGB := termenv.ConvertToRGB(termenvBG)
		fgRGB := termenv.ConvertToRGB(termenvFG)
		_, _, bgL := bgRGB.Hsl()
		_, _, fgL := fgRGB.Hsl()
		fmt.Printf("  FG/BG lightness comparison: BG=%.2f FG=%.2f\n", bgL, fgL)
		fmt.Printf("    Alternative detection (BG < FG): %s background\n", darkOrLight(bgL < fgL))
	}

	// COLORFGBG parsing
	colorfgbg := os.Getenv("COLORFGBG")
	if colorfgbg != "" {
		fmt.Printf("  COLORFGBG: %s\n", colorfgbg)
		parts := strings.Split(colorfgbg, ";")
		if len(parts) >= 2 {
			// Common convention: fg;bg or fg;bright;bg
			bg := parts[len(parts)-1]
			// Values 0-7 are dark, 8-15 are bright
			// 0=black, 7=white, 15=bright white
			fmt.Printf("    Background index: %s\n", bg)
			fmt.Printf("    (0=black, 7=white, 15=bright white; 0-6,8=dark, 7,15=light)\n")
		}
	} else {
		fmt.Println("  COLORFGBG: (not set)")
	}

	fmt.Println()
}

func calculateLuminance(r, g, b float64) float64 {
	// WCAG relative luminance
	rLin := linearize(r)
	gLin := linearize(g)
	bLin := linearize(b)
	return 0.2126*rLin + 0.7152*gLin + 0.0722*bLin
}

func linearize(v float64) float64 {
	if v <= 0.03928 {
		return v / 12.92
	}
	return pow((v+0.055)/1.055, 2.4)
}

func pow(base, exp float64) float64 {
	// Simple power function for small exponents
	if exp == 2.4 {
		// Approximate 2.4 power: x^2 * x^0.4
		return base * base * powFrac(base, 0.4)
	}
	return base
}

func powFrac(base, exp float64) float64 {
	// Newton's method approximation for fractional powers
	// x^0.4 ≈ x^(2/5)
	if base <= 0 {
		return 0
	}
	// Use ln/exp approximation: x^n = e^(n*ln(x))
	// Simplified: for small bases, use linear interpolation
	return 1.0 + exp*(base-1.0) + 0.5*exp*(exp-1)*(base-1.0)*(base-1.0)
}

func darkOrLight(isDark bool) string {
	if isDark {
		return "DARK"
	}
	return "LIGHT"
}

func printRawTerminalQuery() {
	fmt.Println("--- Raw Terminal Query Test ---")
	fmt.Println("  (Directly querying terminal via OSC escape sequences)")

	// Check if we can make terminal raw
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Printf("  Cannot make terminal raw: %v\n", err)
		fmt.Println("  (This is normal in non-interactive contexts like pipes)")
		return
	}
	defer term.Restore(fd, oldState)

	// Query both colors and read all responses
	// Send OSC 11 (background) then OSC 10 (foreground)
	fmt.Fprint(os.Stdout, "\x1b]11;?\x07\x1b]10;?\x07")

	// Read responses with timeout - may get multiple responses
	// Use longer timeout (1s) for slow terminals like Ghostty
	responses := readOSCResponses(fd, 2, 1000*time.Millisecond)

	term.Restore(fd, oldState)

	// Parse and display responses
	var bgResponse, fgResponse string
	for _, resp := range responses {
		printable := makePrintable([]byte(resp))
		if strings.Contains(resp, "]11;") {
			bgResponse = resp
			fmt.Printf("  OSC 11 (background) response:\n")
			fmt.Printf("    Hex: %s\n", hex.EncodeToString([]byte(resp)))
			fmt.Printf("    Printable: %s\n", printable)
		} else if strings.Contains(resp, "]10;") {
			fgResponse = resp
			fmt.Printf("  OSC 10 (foreground) response:\n")
			fmt.Printf("    Hex: %s\n", hex.EncodeToString([]byte(resp)))
			fmt.Printf("    Printable: %s\n", printable)
		} else {
			fmt.Printf("  Unknown response:\n")
			fmt.Printf("    Hex: %s\n", hex.EncodeToString([]byte(resp)))
			fmt.Printf("    Printable: %s\n", printable)
		}

		// Extract and parse RGB
		if idx := strings.Index(resp, "rgb:"); idx != -1 {
			colorPart := resp[idx:]
			endIdx := strings.IndexAny(colorPart, "\x1b\x07")
			if endIdx > 0 {
				colorPart = colorPart[:endIdx]
			}
			fmt.Printf("    Parsed: %s\n", colorPart)
			if r, g, b, ok := parseRGBResponse(colorPart); ok {
				fmt.Printf("    RGB values: R=%d G=%d B=%d (hex: #%02x%02x%02x)\n", r, g, b, r, g, b)
			}
		}
		fmt.Println()
	}

	if bgResponse == "" {
		fmt.Println("  OSC 11 (background): no response received")
	}
	if fgResponse == "" {
		fmt.Println("  OSC 10 (foreground): no response received")
	}
}

// readOSCResponses reads multiple OSC responses from the terminal
func readOSCResponses(fd int, expectedCount int, timeout time.Duration) []string {
	var allData []byte
	buf := make([]byte, 512)
	deadline := time.Now().Add(timeout)

	// Keep reading until we have expected responses or timeout
readLoop:
	for time.Now().Before(deadline) {
		// Set up read with mini-timeout
		dataChan := make(chan int, 1)
		go func() {
			n, _ := os.Stdin.Read(buf)
			dataChan <- n
		}()

		remaining := time.Until(deadline)
		if remaining < 100*time.Millisecond {
			remaining = 100 * time.Millisecond
		}

		select {
		case n := <-dataChan:
			if n > 0 {
				allData = append(allData, buf[:n]...)
				// Check if we have enough responses
				responses := splitOSCResponses(string(allData))
				if len(responses) >= expectedCount {
					return responses
				}
			}
		case <-time.After(remaining):
			// Timeout waiting for more data
			break readLoop
		}
	}

	return splitOSCResponses(string(allData))
}

// splitOSCResponses splits a buffer containing multiple OSC responses
func splitOSCResponses(data string) []string {
	var responses []string
	remaining := data

	for len(remaining) > 0 {
		// Find start of OSC sequence (ESC ])
		start := strings.Index(remaining, "\x1b]")
		if start == -1 {
			break
		}

		// Find end - either BEL (\x07) or ST (\x1b\)
		searchFrom := start + 2
		endBEL := strings.Index(remaining[searchFrom:], "\x07")
		endST := strings.Index(remaining[searchFrom:], "\x1b\\")

		var end int
		var endLen int
		if endBEL >= 0 && (endST < 0 || endBEL <= endST) {
			end = searchFrom + endBEL
			endLen = 1
		} else if endST >= 0 {
			end = searchFrom + endST
			endLen = 2
		} else {
			// No terminator found, take the rest
			responses = append(responses, remaining[start:])
			break
		}

		responses = append(responses, remaining[start:end+endLen])
		remaining = remaining[end+endLen:]
	}

	return responses
}

func makePrintable(data []byte) string {
	var sb strings.Builder
	for _, b := range data {
		if b >= 32 && b < 127 {
			sb.WriteByte(b)
		} else {
			fmt.Fprintf(&sb, "\\x%02x", b)
		}
	}
	return sb.String()
}

func parseRGBResponse(s string) (r, g, b uint16, ok bool) {
	// Format: rgb:RRRR/GGGG/BBBB (16-bit each) or rgb:RR/GG/BB (8-bit)
	s = strings.TrimPrefix(s, "rgb:")
	parts := strings.Split(s, "/")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}

	var vals [3]uint16
	for i, part := range parts {
		var v uint64
		_, err := fmt.Sscanf(part, "%x", &v)
		if err != nil {
			return 0, 0, 0, false
		}
		// Normalize to 8-bit
		switch len(part) {
		case 2:
			vals[i] = uint16(v)
		case 4:
			vals[i] = uint16(v >> 8)
		default:
			vals[i] = uint16(v)
		}
	}

	return vals[0], vals[1], vals[2], true
}
