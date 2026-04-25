package scanner

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dipankardas011/infai/model"
)

func isMmproj(name string) bool {
	return strings.Contains(strings.ToLower(name), "mmproj")
}

func stem(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
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
				out = append(out, model.ModelEntry{
					ScanDir:     dir,
					DirName:     e.Name(),
					GGUFPath:    path,
					MmprojPath:  mmproj,
					DisplayName: e.Name() + " / " + stem(filepath.Base(path)),
				})
			}
		}
	}
	return out, nil
}
