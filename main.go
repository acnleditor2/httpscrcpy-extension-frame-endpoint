package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"
)

type portState struct {
	ffmpeg              *exec.Cmd
	ffmpegStdin         io.WriteCloser
	ffmpegStdout        io.ReadCloser
	contentLengthString string
	widthString         string
	heightString        string
	frame               []byte
	m                   sync.Mutex
}

var portMap = map[int]*portState{}

func main() {
	if len(os.Args) == 4 || (len(os.Args) == 5 && os.Args[4] == "noalpha") {
		var err error

		{
			var b bytes.Buffer
			b.WriteByte(byte(len(os.Args[1])))
			b.WriteString(os.Args[1])
			b.WriteByte(1)
			b.WriteByte(byte(len(os.Args[2])))
			b.WriteString(os.Args[2])

			_, err = b.WriteTo(os.Stdout)
			if err != nil {
				panic(err)
			}
		}

		var data []byte
		var n int

		for {
			data = make([]byte, 3)

			n, err = io.ReadFull(os.Stdin, data)
			if err != nil {
				panic(err)
			}
			if n != 3 {
				break
			}

			port := int(binary.NativeEndian.Uint16(data[1:]))

			switch data[0] {
			case 0:
				data = make([]byte, 4)

				n, err = io.ReadFull(os.Stdin, data)
				if err != nil {
					panic(err)
				}
				if n != 4 {
					break
				}

				queryCount := int(binary.NativeEndian.Uint32(data))

				for i := 0; i < queryCount; i++ {
					data = make([]byte, 4)

					n, err = io.ReadFull(os.Stdin, data)
					if err != nil {
						panic(err)
					}
					if n != 4 {
						return
					}

					data = make([]byte, int(binary.NativeEndian.Uint32(data)))

					n, err = io.ReadFull(os.Stdin, data)
					if err != nil {
						panic(err)
					}
					if n != len(data) {
						return
					}

					data = make([]byte, 4)

					n, err = io.ReadFull(os.Stdin, data)
					if err != nil {
						panic(err)
					}
					if n != 4 {
						return
					}

					data = make([]byte, int(binary.NativeEndian.Uint32(data)))

					n, err = io.ReadFull(os.Stdin, data)
					if err != nil {
						panic(err)
					}
					if n != len(data) {
						return
					}
				}

				data = make([]byte, 4)

				n, err = io.ReadFull(os.Stdin, data)
				if err != nil {
					panic(err)
				}
				if n != 4 {
					break
				}

				headerCount := int(binary.NativeEndian.Uint32(data))

				for i := 0; i < headerCount; i++ {
					data = make([]byte, 4)

					n, err = io.ReadFull(os.Stdin, data)
					if err != nil {
						panic(err)
					}
					if n != 4 {
						return
					}

					data = make([]byte, int(binary.NativeEndian.Uint32(data)))

					n, err = io.ReadFull(os.Stdin, data)
					if err != nil {
						panic(err)
					}
					if n != len(data) {
						return
					}

					data = make([]byte, 4)

					n, err = io.ReadFull(os.Stdin, data)
					if err != nil {
						panic(err)
					}
					if n != 4 {
						return
					}

					data = make([]byte, int(binary.NativeEndian.Uint32(data)))

					n, err = io.ReadFull(os.Stdin, data)
					if err != nil {
						panic(err)
					}
					if n != len(data) {
						return
					}
				}

				ps, ok := portMap[port]

				var b bytes.Buffer
				if ok {
					binary.Write(&b, binary.NativeEndian, uint16(200))
					b.WriteByte(6)
					b.WriteByte(13)
					b.WriteString("Cache-Control")
					b.WriteByte(8)
					b.WriteString("no-store")
					ps.m.Lock()
					b.WriteByte(14)
					b.WriteString("Content-Length")
					b.WriteByte(byte(len(ps.contentLengthString)))
					b.WriteString(ps.contentLengthString)
					b.WriteByte(12)
					b.WriteString("Content-Type")
					b.WriteByte(24)
					b.WriteString("application/octet-stream")
					b.WriteByte(5)
					b.WriteString("Width")
					b.WriteByte(byte(len(ps.widthString)))
					b.WriteString(ps.widthString)
					b.WriteByte(6)
					b.WriteString("Height")
					b.WriteByte(byte(len(ps.heightString)))
					b.WriteString(ps.heightString)
					b.WriteByte(8)
					b.WriteString("Channels")
					b.WriteByte(1)
					if len(os.Args) == 4 {
						b.WriteString("4")
					} else {
						b.WriteString("3")
					}
					binary.Write(&b, binary.NativeEndian, uint32(len(ps.frame)))
					b.Write(ps.frame)
					ps.m.Unlock()
					binary.Write(&b, binary.NativeEndian, uint32(0))
					b.WriteByte(0)
				} else {
					binary.Write(&b, binary.NativeEndian, uint16(404))
					b.WriteByte(0)
					binary.Write(&b, binary.NativeEndian, uint32(0))
					b.WriteByte(0)
				}

				_, err = b.WriteTo(os.Stdout)
				if err != nil {
					panic(err)
				}
			case 1:
				data = make([]byte, 1)

				n, err = io.ReadFull(os.Stdin, data)
				if err != nil {
					panic(err)
				}
				if n != 1 {
					return
				}

				data = make([]byte, int(data[0])+12)

				n, err = io.ReadFull(os.Stdin, data)
				if err != nil {
					panic(err)
				}
				if n != len(data) {
					return
				}

				ps, ok := portMap[port]
				if !ok {
					portMap[port] = &portState{}
					ps = portMap[port]
				}

				width := int(binary.NativeEndian.Uint32(data[len(data)-8:]))
				height := int(binary.NativeEndian.Uint32(data[len(data)-4:]))
				if len(os.Args) == 4 {
					ps.frame = make([]byte, width*height*4)
				} else {
					ps.frame = make([]byte, width*height*3)
				}
				ps.contentLengthString = strconv.Itoa(len(ps.frame))
				ps.widthString = strconv.Itoa(width)
				ps.heightString = strconv.Itoa(height)

				if ps.ffmpeg != nil {
					ps.ffmpeg.Process.Kill()
					ps.ffmpeg.Wait()
				}

				ps.ffmpeg = exec.Command(
					os.Args[3],
					"-probesize",
					"32",
					"-analyzeduration",
					"0",
					"-re",
					"-f",
					map[uint32]string{
						0x68323634: "h264",
						0x68323635: "hevc",
						0x617631:   "av1",
					}[binary.NativeEndian.Uint32(data[len(data)-12:])],
					"-i",
					"-",
					"-f",
					"rawvideo",
					"-pix_fmt",
					map[int]string{
						4: "rgba",
						5: "rgb24",
					}[len(os.Args)],
					"-vf",
					fmt.Sprintf("scale='min(%[1]d,iw)':'min(%[2]d,ih)':force_original_aspect_ratio=decrease,pad=%[1]d:%[2]d:-1:-1:color=black", width, height),
					"-",
				)

				ps.ffmpeg.Stderr = os.Stderr

				ps.ffmpegStdin, err = ps.ffmpeg.StdinPipe()
				if err != nil {
					panic(err)
				}

				ps.ffmpegStdout, err = ps.ffmpeg.StdoutPipe()
				if err != nil {
					panic(err)
				}

				err = ps.ffmpeg.Start()
				if err != nil {
					panic(err)
				}

				go func(p int) {
					ps := portMap[p]
					frame := make([]byte, len(ps.frame))

					for {
						n, err := io.ReadFull(ps.ffmpegStdout, frame)
						if err != nil {
							break
						}
						if n != len(frame) {
							break
						}

						ps.m.Lock()
						copy(ps.frame, frame)
						ps.m.Unlock()
					}
				}(port)
			case 2:
				data = make([]byte, 12)

				n, err = io.ReadFull(os.Stdin, data)
				if err != nil {
					panic(err)
				}
				if n != 12 {
					return
				}

				data = make([]byte, int(binary.BigEndian.Uint32(data[8:])))

				n, err = io.ReadFull(os.Stdin, data)
				if err != nil {
					panic(err)
				}
				if n != len(data) {
					return
				}

				ps, ok := portMap[port]

				if !ok {
					return
				}

				n, err = ps.ffmpegStdin.Write(data)
				if n < len(data) {
					return
				}
			default:
				return
			}
		}
	}
}
