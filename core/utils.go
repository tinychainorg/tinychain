package core

import (
	"fmt"
	"log"
	"os"

	"github.com/fatih/color"
)

func NewLogger(prefix string) *log.Logger {
	// 2024/06/30 00:56:06 [prefix] message
	return log.New(os.Stdout, color.HiGreenString(fmt.Sprintf("[%s] ", prefix)), log.Ldate|log.Ltime|log.Lmsgprefix)
}
