package commonutil

import (
	"fmt"
	"runtime"
	"time"
)

// TimeConsume ...
func TimeConsume(start time.Time) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return
	}

	// get Fun object from pc
	funcName := runtime.FuncForPC(pc).Name()
	fmt.Println(funcName, "cost:", time.Since(start).String())
}
