package ngcache

import (
	"bytes"
	"text/template"
	"io"
	"path/filepath"

	"github.com/omeid/slurp"
)

type Config struct {
	Name       string //Name of the generated file.
	Module     string //Name of AngularJS module.
	Standalone bool   //Create a new AngularJS module, instead of using an existing.
}

func readall(r io.Reader) (string, error) {
	var buff bytes.Buffer
	_, err := buff.ReadFrom(r)
	return buff.String(), err
}

var cacheTemplate = template.Must(template.New("").Funcs(template.FuncMap{"readall": readall}).Parse(`angular.module("{{ .Module }}"{{ if .Standalone }} , [] {{ end }}).run(["$templateCache", function($templateCache) { {{ range $path, $file := .Files }} 
$templateCache.put("{{ $path }}", "{{ readall $file | js }}"); {{ end }}
}])`))

type cache struct {
	Config
	Files map[string]slurp.File
}

// A build stage creates a new build and adds all the files coming through the channel to
// the Build and returns the result of Build as a File on the output channel.
func Build(c *slurp.C, config Config) slurp.Stage {
	return func(in <-chan slurp.File, out chan<- slurp.File) {

		b := cache{config, make(map[string]slurp.File)}

		for file := range in {
			path, _ := filepath.Rel(file.Dir, file.Path)
			path = filepath.ToSlash(path)
			c.Infof("Adding %s", path)
			b.Files[path] = file
			defer file.Close() //Close files AFTER we have build our package.
		}

		buff := new(bytes.Buffer)
		err := cacheTemplate.Execute(buff, b)
		if err != nil {
			c.Error(err)
			return
		}

		stat := slurp.FileInfo{}
		stat.SetName(b.Name)
		stat.SetSize(int64(buff.Len()))

		sf := slurp.File{
			Reader: buff,
			Path:   b.Name,
		}

		sf.SetStat(&stat)

		out <- sf
	}
}
