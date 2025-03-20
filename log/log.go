package axlog

import (
	"fmt"
	"os"
)

func Logf(format string, a ...any) {
	if os.Getenv("ENV_MODE") == "dev" {
		fmt.Printf(format, a...)
	}
}

func Logln(a ...any) {
	if os.Getenv("ENV_MODE") == "dev" {
		fmt.Println(a...)
	}
}

func Loglf(format string, a ...any) {
	if os.Getenv("ENV_MODE") == "dev" {
		fmt.Printf(format+"\n", a...)
	}
}
