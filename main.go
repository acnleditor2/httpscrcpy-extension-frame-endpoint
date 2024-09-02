package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
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

type Config struct {
	ID        string         `json:"id"`
	Ffmpeg    string         `json:"ffmpeg"`
	Alpha     bool           `json:"alpha"`
	Endpoints map[string]int `json:"endpoints"`
}

func main() {
	if len(os.Args) == 2 {
		var config Config

		err := json.Unmarshal([]byte(os.Args[1]), &config)
		if err != nil {
			panic(err)
		}

		var path string

		{
			var b bytes.Buffer
			b.WriteByte(byte(len(config.ID)))
			b.WriteString(config.ID)
			b.WriteByte(byte(len(config.Endpoints)))
			for endpoint := range config.Endpoints {
				b.WriteByte(byte(len(endpoint)))
				b.WriteString(endpoint)

				if len(config.Endpoints) == 1 {
					path = endpoint
				}
			}

			_, err = b.WriteTo(os.Stdout)
			if err != nil {
				panic(err)
			}
		}

		portMap := map[int]*portState{}
		var data []byte
		var n int

		for {
			data = make([]byte, 1)

			n, err = io.ReadFull(os.Stdin, data)
			if err != nil {
				panic(err)
			}
			if n != 1 {
				break
			}

			switch data[0] {
			case 0:
				if len(config.Endpoints) > 1 {
					data = make([]byte, 1)

					n, err = io.ReadFull(os.Stdin, data)
					if err != nil {
						panic(err)
					}
					if n != 1 {
						return
					}

					data = make([]byte, int(data[0]))

					n, err = io.ReadFull(os.Stdin, data)
					if err != nil {
						panic(err)
					}
					if n != len(data) {
						return
					}

					path = string(data)
				}

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

				ps, ok := portMap[config.Endpoints[path]]

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
					if config.Alpha {
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
					b.WriteByte(1)
					b.WriteByte(13)
					b.WriteString("Cache-Control")
					b.WriteByte(8)
					b.WriteString("no-store")
					binary.Write(&b, binary.NativeEndian, uint32(0))
					b.WriteByte(0)
				}

				_, err = b.WriteTo(os.Stdout)
				if err != nil {
					panic(err)
				}
			case 1:
				data = make([]byte, 3)

				n, err = io.ReadFull(os.Stdin, data)
				if err != nil {
					panic(err)
				}
				if n != 3 {
					return
				}

				port := int(binary.NativeEndian.Uint16(data[:2]))
				data = make([]byte, int(data[2])+12)

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
				if config.Alpha {
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
					config.Ffmpeg,
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
					map[bool]string{
						false: "rgb24",
						true:  "rgba",
					}[config.Alpha],
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

				go func() {
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
				}()
			case 2:
				data = make([]byte, 14)

				n, err = io.ReadFull(os.Stdin, data)
				if err != nil {
					panic(err)
				}
				if n != 14 {
					return
				}

				port := int(binary.NativeEndian.Uint16(data[:2]))
				data = make([]byte, int(binary.BigEndian.Uint32(data[10:])))

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
