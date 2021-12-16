package bwlimit

import (
	"errors"
	"io"
	"log"
	"time"
)

const (
	B  = 8
	KB = B * 1024
	MB = KB * 1024
	GB = MB * 1024
)

var (
	Debug = false
)

func Copy(dst io.Writer, src io.Reader, bps uint64) (written int64, err error) {
	size := 1024
	threshold := bps / 8
	if bps/8 < 1024 { // Bps < 1KB/s
		threshold = 1
		size = 1
	}

	buf := make([]byte, size)

	recent_written := uint64(0) // Bytes
	recent_written_time := time.Now()
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errors.New("invalid write result")
				}
			}
			written += int64(nw)
			recent_written += uint64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
			if recent_written > threshold {
				for {
					if uint64(time.Since(recent_written_time)) == 0 {
						continue
					}
					if recent_written*uint64(8*time.Second/time.Since(recent_written_time)) < bps {
						break
					}
					time.Sleep(time.Millisecond * 1)
				}
				if Debug {
					log.Println("speed:", recent_written*uint64(8*time.Second/time.Since(recent_written_time)), "bps")
				}
				recent_written = 0
				recent_written_time = time.Now()
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
