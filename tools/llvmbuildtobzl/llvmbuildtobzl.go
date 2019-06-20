/*
 * Copyright 2019 The Kythe Authors. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bytes"
	"flag"
	"log"
	"os"
	"regexp"
	"strings"

	"bitbucket.org/creachadair/stringset"
	"github.com/creachadair/ini"
	"github.com/kythe/llvmbzlgen/path"
	"github.com/kythe/llvmbzlgen/writer"
)

var (
	camelPattern = regexp.MustCompile("(.)([A-Z][a-z]+)")
	stringProps  = stringset.New("name", "parent", "library_name")
	listProps    = stringset.New("dependencies", "required_libraries", "add_to_library_groups")
)

type iniFile map[string]iniSection
type iniSection map[string][]string
type propArgs map[string]interface{}

func load(path string) (iniFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	f := make(iniFile)
	return f, ini.Parse(file, ini.Handler{
		Section: func(_ ini.Location, name string) error {
			f[name] = make(iniSection)
			return nil
		},
		KeyValue: func(loc ini.Location, key string, values []string) error {
			f[loc.Section][key] = flatSplit(values)
			return nil
		},
	})
}

func (f *iniFile) Subdirectories() []string {
	if section := f.Section("common"); section != nil {
		return section.Key("subdirectories")
	}
	return nil
}

func (f *iniFile) Components() []iniSection {
	var result []iniSection
	for name, section := range *f {
		if strings.HasPrefix(name, "component_") {
			result = append(result, section)
		}
	}
	return result
}

func (f *iniFile) Section(name string) *iniSection {
	s, ok := (*f)[name]
	if !ok {
		return nil
	}
	return &s
}

func (s *iniSection) Key(name string) []string {
	k, ok := (*s)[name]
	if !ok {
		return nil
	}
	return k
}

func (s *iniSection) RuleKind() string {
	if t := s.Key("type"); len(t) == 1 {
		return strings.ToLower(camelPattern.ReplaceAllString(t[0], "${1}_${2}"))
	}
	return ""
}

func (s *iniSection) Properties() propArgs {
	result := make(map[string]interface{}, len(*s)-1)
	for k, v := range *s {
		switch {
		case listProps.Contains(k):
			result[k] = v
		case stringProps.Contains(k):
			result[k] = v[0]
		}
	}
	return result
}

func (pa propArgs) MarshalStarlark() ([]byte, error) {
	var b bytes.Buffer
	i := 0
	for k, v := range pa {
		if i > 0 {
			b.WriteString(", ")
		}
		i++
		b.WriteString(k)
		b.WriteString(" = ")
		val, err := writer.Marshal(v)
		if err != nil {
			return nil, err
		}
		b.Write(val)
	}
	return b.Bytes(), nil
}

func flatSplit(values []string) []string {
	var result []string
	for _, v := range values {
		result = append(result, strings.Split(v, " ")...)
	}
	return result
}

type visitor struct {
	w *writer.StarlarkWriter
}

func (v visitor) visitBuildFile(dir path.Path) ([]path.Path, func() error, error) {
	file, err := load(path.JoinString(dir, "LLVMBuild.txt").String())
	if err != nil {
		return nil, nil, err
	}
	for _, s := range file.Components() {
		if err := v.w.WriteCommand(s.RuleKind(), s.Properties()); err != nil {
			return nil, nil, err
		}
	}
	return path.ToPaths(file.Subdirectories()), nil, nil
}

func (v visitor) Start() path.PathVisitor {
	if err := v.w.BeginMacro("generated_llvm_build_targets"); err != nil {
		log.Fatal(err)
	}
	return v.visitBuildFile
}

func (v visitor) End() error {
	return v.w.EndMacro()
}

func main() {
	flag.Parse()
	v := visitor{writer.NewStarlarkWriter(os.Stdout)}
	if err := path.WalkPath(path.New(flag.Args()[0]), v.Start()); err != nil {
		log.Fatal(err)
	}
	if err := v.End(); err != nil {
		log.Fatal(err)
	}
}
