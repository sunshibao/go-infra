package stackutil

import (
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

type logger interface {
	Print(msg string)
}

type FmtLogger struct {
}

func (FmtLogger) Print(msg string) {
	fmt.Println(msg)
}

type ZapLogger struct {
	Logger *zap.Logger
}

func (l ZapLogger) Print(msg string) {
	l.Logger.Info(msg)
}

const stack_dump_signal = syscall.SIGUSR1

func SetupStackDumper(logger logger) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, stack_dump_signal)
	go func() {
		for range c {
			dumpStacks(logger)
		}
	}()
	logger.Print(fmt.Sprintf("succeed to setup a stack dumper with signal:%d", stack_dump_signal))
}

func dumpStacks(logger logger) {
	buf := make([]byte, 32768)
	buf = buf[:runtime.Stack(buf, true)]
	logger.Print(fmt.Sprintf("=== BEGIN goroutine stack dump ===\n%s\n=== END goroutine stack dump ===", buf))
}
