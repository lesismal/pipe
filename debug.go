package pipe

import (
	"log"
	"runtime"
	"unsafe"
)

func Recover() {
	if err := recover(); err != nil {
		const size = 64 << 10
		buf := make([]byte, size)
		buf = buf[:runtime.Stack(buf, false)]
		log.Printf("execute failed: %v\n%v\n", err, *(*string)(unsafe.Pointer(&buf)))
	}
}
