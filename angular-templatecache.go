package ngcache

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"text/template"

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
	fmt.Println(buff.String())
	return buff.String(), err
}

var cacheTemplate = template.Must(template.New("").Funcs(template.FuncMap{"readall": readall}).Parse(`angular.module("{{ .Module }}"{{ if .Standalone }} , [] {{ end }}).run(["$templateCache", function($templateCache) { {{ range $path, $file := .Files }} 
$templateCache.put("{{ $path }}", "{{ $file | js }}"); {{ end }}
}])`))

type cache struct {
	Config
	Files map[string]*bytes.Buffer
}

// A build stage creates a new build and adds all the files coming through the channel to
// the Build and returns the result of Build as a File on the output channel.
func Build(c *slurp.C, config Config) slurp.Stage {
	return func(in <-chan slurp.File, out chan<- slurp.File) {

		b := cache{config, make(map[string]*bytes.Buffer)}

		for file := range in {
			path, _ := filepath.Rel(file.Dir, file.Path)
			path = filepath.ToSlash(path)
			c.Infof("Adding %s", path)
			buff := new(bytes.Buffer)
			_, err := buff.ReadFrom(file)
			if err != nil {
				c.Error(err)
			}
			b.Files[path] = buff
			file.Close() //Close files AFTER we have build our package.
		}

		buff := new(bytes.Buffer)
		err := cacheTemplate.Execute(buff, b)
		if err != nil {
			c.Error(err)
			return
		}

		sf := slurp.File{
			Reader: buff,
			Path:   b.Name,
		}
		sf.FileInfo.SetName(b.Name)
		sf.FileInfo.SetSize(int64(buff.Len()))

		out <- sf
	}
}
