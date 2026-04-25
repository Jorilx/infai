package scanner

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dipankardas011/infai/model"
)

const (
	GGUF_MAGIC       = 0x46554747
	GGUF_VERSION     = 3
	KEY_GENERAL_ARCH = "general.architecture"
	KEY_GENERAL_NAME = "general.name"
	KEY_VISION       = "vision"
)

type ggufMetadata struct {
	Architecture string
	ModelName    string
	IsVision     bool
}

type readSeeker struct {
	r   io.Reader
	pos int64
}

func (rs *readSeeker) Read(p []byte) (n int, err error) {
	n, err = rs.r.Read(p)
	rs.pos += int64(n)
	return n, err
}

func (rs *readSeeker) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = rs.pos + offset
	case io.SeekEnd:
		f, ok := rs.r.(*os.File)
		if !ok {
			return 0, errors.New("seek from end requires *os.File")
		}
		info, err := f.Stat()
		if err != nil {
			return 0, err
		}
		newPos = info.Size() + offset
	default:
		return 0, errors.New("invalid seek whence")
	}
	if newPos < 0 {
		return 0, errors.New("negative seek position")
	}
	rs.pos = newPos
	return newPos, nil
}

func ParseGGUFMetadata(path string) (*ggufMetadata, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := &readSeeker{r: f}

	var magic uint32
	if err := binary.Read(reader, binary.LittleEndian, &magic); err != nil {
		return nil, err
	}
	if magic != GGUF_MAGIC {
		return nil, errors.New("invalid GGUF magic")
	}

	var version uint32
	if err := binary.Read(reader, binary.LittleEndian, &version); err != nil {
		return nil, err
	}
	if version > GGUF_VERSION {
		return nil, errors.New("unsupported GGUF version")
	}

	var tensorCount uint64
	if err := binary.Read(reader, binary.LittleEndian, &tensorCount); err != nil {
		return nil, err
	}

	var metadataKVCount uint32
	if err := binary.Read(reader, binary.LittleEndian, &metadataKVCount); err != nil {
		return nil, err
	}

	metadata := &ggufMetadata{}

	_, err = readAlignment(reader)
	if err != nil {
		return nil, err
	}

	for i := uint32(0); i < metadataKVCount; i++ {
		key, err := readString(reader)
		if err != nil {
			return nil, err
		}

		valueType, err := readUint32(reader)
		if err != nil {
			return nil, err
		}

		switch valueType {
		case 0:
			val, err := readUint32(reader)
			if err != nil {
				return nil, err
			}
			_ = val
		case 1:
			val, err := readUint64(reader)
			if err != nil {
				return nil, err
			}
			_ = val
		case 2:
			val, err := readFloat32(reader)
			if err != nil {
				return nil, err
			}
			_ = val
		case 3:
			val, err := readFloat64(reader)
			if err != nil {
				return nil, err
			}
			_ = val
		case 4:
			val, err := readString(reader)
			if err != nil {
				return nil, err
			}
			if key == KEY_GENERAL_ARCH {
				metadata.Architecture = val
			} else if key == KEY_GENERAL_NAME {
				metadata.ModelName = val
			}
		case 5:
			arr, err := readArray(reader)
			if err != nil {
				return nil, err
			}
			_ = arr
		case 6:
			arr, err := readStringArray(reader)
			if err != nil {
				return nil, err
			}
			_ = arr
		case 7:
			val, err := readUint64(reader)
			if err != nil {
				return nil, err
			}
			_ = val
		default:
			return nil, errors.New("unknown GGUF value type")
		}

		reader.Seek(0, io.SeekCurrent)
	}

	if strings.Contains(strings.ToLower(metadata.Architecture), KEY_VISION) {
		metadata.IsVision = true
	}

	return metadata, nil
}

func readAlignment(rs *readSeeker) (uint32, error) {
	pos, err := rs.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}
	align := 32 - (pos % 32)
	if align == 32 {
		align = 0
	}
	if align > 0 {
		rs.Seek(int64(align), io.SeekCurrent)
	}
	return uint32(align), nil
}

const maxStringLen = 1024 * 1024 // 1MB max string length

func readString(rs *readSeeker) (string, error) {
	len, err := readUint64(rs)
	if err != nil {
		return "", err
	}
	if len > maxStringLen {
		return "", fmt.Errorf("string too large: %d", len)
	}
	data := make([]byte, len)
	rs.Read(data)
	return string(data[:len-1]), nil
}

func readUint32(rs *readSeeker) (uint32, error) {
	var val uint32
	err := binary.Read(rs, binary.LittleEndian, &val)
	return val, err
}

func readUint64(rs *readSeeker) (uint64, error) {
	var val uint64
	err := binary.Read(rs, binary.LittleEndian, &val)
	return val, err
}

func readFloat32(rs *readSeeker) (float32, error) {
	var val float32
	err := binary.Read(rs, binary.LittleEndian, &val)
	return val, err
}

func readFloat64(rs *readSeeker) (float64, error) {
	var val float64
	err := binary.Read(rs, binary.LittleEndian, &val)
	return val, err
}

func readArray(rs *readSeeker) ([]uint32, error) {
	len, err := readUint64(rs)
	if err != nil {
		return nil, err
	}
	arr := make([]uint32, len)
	for i := uint64(0); i < len; i++ {
		val, err := readUint32(rs)
		if err != nil {
			return nil, err
		}
		arr[i] = val
	}
	return arr, nil
}

func readStringArray(rs *readSeeker) ([]string, error) {
	len, err := readUint64(rs)
	if err != nil {
		return nil, err
	}
	arr := make([]string, len)
	for i := uint64(0); i < len; i++ {
		s, err := readString(rs)
		if err != nil {
			return nil, err
		}
		arr[i] = s
	}
	return arr, nil
}

func isMmproj(name string) bool {
	return strings.Contains(strings.ToLower(name), "mmproj")
}

func stem(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}

func ComputeSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// Scan walks one level under each dir in dirs, returning one ModelEntry per non-mmproj .gguf file.
func Scan(dirs []string) ([]model.ModelEntry, error) {
	var out []model.ModelEntry
	seen := map[string]bool{}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			subdir := filepath.Join(dir, e.Name())
			files, err := os.ReadDir(subdir)
			if err != nil {
				continue
			}
			var mmproj string
			var mains []string
			for _, f := range files {
				if f.IsDir() || filepath.Ext(f.Name()) != ".gguf" {
					continue
				}
				if isMmproj(f.Name()) {
					mmproj = filepath.Join(subdir, f.Name())
				} else {
					mains = append(mains, filepath.Join(subdir, f.Name()))
				}
			}
			for _, path := range mains {
				if seen[path] {
					continue
				}
				seen[path] = true

				checksum, _ := ComputeSHA256(path)
				metadata, _ := ParseGGUFMetadata(path)

				arch := ""
				modelName := ""
				if metadata != nil {
					arch = metadata.Architecture
					modelName = metadata.ModelName
				}

				if mmproj == "" && metadata != nil && metadata.IsVision {
					for _, f := range files {
						if isMmproj(f.Name()) {
							mmproj = filepath.Join(subdir, f.Name())
						}
					}
				}

				displayName := e.Name() + " / " + stem(filepath.Base(path))
				if modelName != "" {
					displayName = modelName
				}

				out = append(out, model.ModelEntry{
					ScanDir:      dir,
					DirName:      e.Name(),
					GGUFPath:     path,
					MmprojPath:   mmproj,
					DisplayName:  displayName,
					Checksum:     checksum,
					Architecture: arch,
					ModelName:    modelName,
				})
			}
		}
	}
	return out, nil
}
