package util

import (
	"fmt"
	"os"
)

const (
	ColorReset   = "\033[0m"
	ColorRed     = "\033[31m"
	ColorGreen   = "\033[32m"
	ColorYellow  = "\033[33m"
	ColorBlue    = "\033[34m"
	ColorMagenta = "\033[35m"
	ColorCyan    = "\033[36m"
	ColorWhite   = "\033[37m"
	ColorBold    = "\033[1m"
)

// PrintGreen prints text in green color
func PrintGreen(format string, a ...any) {
	fmt.Printf(ColorGreen+format+ColorReset+"\n", a...)
}

// PrintYellow prints text in yellow color
func PrintYellow(format string, a ...any) {
	fmt.Printf(ColorYellow+format+ColorReset+"\n", a...)
}

// PrintRed prints text in red color
func PrintRed(format string, a ...any) {
	fmt.Printf(ColorRed+format+ColorReset+"\n", a...)
}

// PrintCyan prints text in cyan/skyBlue color
func PrintCyan(format string, a ...any) {
	fmt.Printf(ColorCyan+ColorBold+format+ColorReset+"\n", a...)
}

// PrintInfo prints standard info log
func PrintInfo(msg string) {
	fmt.Fprintf(os.Stderr, ColorCyan+" [INFO] "+ColorReset+"%s\n", msg)
}

// PrintSuccess prints standard success log
func PrintSuccess(msg string) {
	fmt.Fprintf(os.Stderr, ColorGreen+" [SUCCESS] "+ColorReset+"%s\n", msg)
}

// PrintWarning prints standard warning log
func PrintWarning(msg string) {
	fmt.Fprintf(os.Stderr, ColorYellow+" [WARN] "+ColorReset+"%s\n", msg)
}

// PrintError prints standard error log
func PrintError(msg string) {
	fmt.Fprintf(os.Stderr, ColorRed+" [ERROR] "+ColorReset+"%s\n", msg)
}

// PrintStep prints progress step format
func PrintStep(current, total int, title string) {
	fmt.Fprintf(os.Stderr, "\n"+ColorCyan+ColorBold+" ──> 进度 %d/%d : %s"+ColorReset+"\n", current, total, title)
}

// PrintDivider prints a decorative line
func PrintDivider() {
	fmt.Fprintln(os.Stderr, ColorYellow + "==============================================================" + ColorReset)
}

// FatalError prints error and terminates execution
func FatalError(format string, a ...any) {
	PrintError(fmt.Sprintf(format, a...))
	os.Exit(1)
}
