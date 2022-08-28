package runner3

import (
	"fmt"
	"log"
	"os"
)

// getColorCode returns an escaped color code sequence for use in log.Print... funcs.
// If an unknown color is given, default to white text since most terminals have a
// dark background.
//
// I will admit I do not understand exactly what the Sprintf calls are doing. I know
// it is an escape sequence, that is all.
func getColorCode(color string) string {
	switch color {
	case "reset":
		return fmt.Sprintf("\033[%sm", "0")
	case "black":
		return fmt.Sprintf("\033[%sm", "30")
	case "red":
		return fmt.Sprintf("\033[%sm", "31")
	case "green":
		return fmt.Sprintf("\033[%sm", "32")
	case "yellow":
		return fmt.Sprintf("\033[%sm", "33")
	case "blue":
		return fmt.Sprintf("\033[%sm", "34")
	case "magenta":
		return fmt.Sprintf("\033[%sm", "35")
	case "cyan":
		return fmt.Sprintf("\033[%sm", "36")
	case "white":
		return fmt.Sprintf("\033[%sm", "37")

	default:
		log.Println("unknown color, defaulting to white")
		return fmt.Sprintf("\033[%sm", "37")
	}
}

// coloredLogger holds information about the logging output such as color and prefix.
type coloredLogger struct {
	prefix    string
	color     string
	colorcode string //populated via newLogger
	resetcode string //populated via newLogger, should be the escaped "0" sequence.
	logger    log.Logger
}

// newLogger returns a populated coloredLogger.
func newLogger(prefix, color string) *coloredLogger {
	return &coloredLogger{
		prefix:    prefix,
		color:     color,
		colorcode: getColorCode(color),
		resetcode: getColorCode("reset"),
		logger:    *log.New(os.Stderr, "", 0),
	}
}

// Print calls log.Printf with the output colorized and prefixed per the coloredLogger's
// settings.
func (c *coloredLogger) Println(v ...any) {
	c.logger.Printf("%s%s | ", c.colorcode, c.prefix)
	c.logger.Println(v...)
	c.logger.Printf(c.resetcode)
}

func (c *coloredLogger) Printf(format string, v ...any) {
	// c.logger.Printf("%s%s | %s%s", c.colorcode, c.prefix, v, c.resetcode)

	format = fmt.Sprintf("%s %s |%s %s", c.colorcode, c.prefix, c.resetcode, format)
	c.logger.Printf(format, v...)
}

func (c *coloredLogger) Fatalln(v ...any) {
	c.logger.Fatalf("%s%s | %s%s", c.colorcode, c.prefix, v, c.resetcode)
}
