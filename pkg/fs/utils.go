package fs

import (
	"fmt"
	"time"
	"unicode"

	"github.com/shirou/gopsutil/process"
)

var TIMEMAX = time.Unix(1<<63-62135596801, 999999999)

func MaxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}

	return b
}

func MinInt64(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
}

func MaxUInt32(a, b uint32) uint32 {
	if a > b {
		return a
	}

	return b
}

func MinUInt32(a, b uint32) uint32 {
	if a < b {
		return a
	}

	return b
}

func MaxUInt64(a, b uint64) uint64 {
	if a > b {
		return a
	}

	return b
}

func MinUInt64(a, b uint64) uint64 {
	if a < b {
		return a
	}

	return b
}

func PBool(v bool) *bool {
	return &v
}

func PInt32(v int32) *int32 {
	return &v
}

func PUInt32(v uint32) *uint32 {
	return &v
}

func PInt64(v int64) *int64 {
	return &v
}

func PUInt64(v uint64) *uint64 {
	return &v
}

func PString(v string) *string {
	return &v
}

func PTime(v time.Time) *time.Time {
	return &v
}

func NilStr(v *string) string {
	if v == nil {
		return ""
	}

	return *v
}

func xattrEscape(value []byte) (s string) {
	for _, c := range value {
		if c == '%' {
			s += "%25"
		} else if unicode.IsPrint(rune(c)) {
			s += string(c)
		} else {
			s += "%" + fmt.Sprintf("%02X", c)
		}
	}

	return
}

func Dup(value []byte) []byte {
	ret := make([]byte, len(value))
	copy(ret, value)
	return ret
}

type empty struct{}

// TODO(dotslash/khc): Remove this semaphore in favor of
// https://godoc.org/golang.org/x/sync/semaphore
type semaphore chan empty

func (sem semaphore) P(n int) {
	for i := 0; i < n; i++ {
		sem <- empty{}
	}
}

func (sem semaphore) V(n int) {
	for i := 0; i < n; i++ {
		<-sem
	}
}

// GetTgid returns the tgid for the given pid.
func GetTgid(pid uint32) (tgid *int32, err error) {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, err
	}
	tgidVal, err := p.Tgid()
	if err != nil {
		return nil, err
	}
	return &tgidVal, nil
}
