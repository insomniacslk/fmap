package fmap

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"

	"github.com/alecthomas/participle"
)

// Section represents a generic flashmap section. This is also used for the text
// parser to read a flashmap file.
type Section struct {
	Name       string     `@Ident`
	Annotation *string    `("(" { @Ident } ")")?`
	Start      *int       `("@" @Int)?`
	Size       int        `@Int`
	Unit       string     `@("k"|"K"|"m"|"M")?`
	Sections   []*Section `("{" { @@ } "}")*`
}

// ToFlashmap returns the text representation of the Section struct.
func (s *Section) ToFlashmap() string {
	return s.Indent("\t", 0)
}

// Indent indents a section with the given prefix string and indentation level.
// This is suitable to print nested sections to be serialized to text file.
func (s *Section) Indent(prefix string, level int) string {
	indent := strings.Repeat(prefix, level)
	ret := indent + s.Name
	if s.Annotation != nil {
		ret += "(" + *s.Annotation + ")"
	}
	if s.Start != nil {
		ret += fmt.Sprintf("@0x%x", *s.Start)
	}
	if s.Unit != "" {
		ret += fmt.Sprintf(" %d%s", s.Size, s.Unit)
	} else {
		ret += fmt.Sprintf(" 0x%x", s.Size)
	}
	if len(s.Sections) > 0 {
		ret += " {\n"
		for _, sec := range s.Sections {
			ret += sec.Indent(prefix, level+1)
		}
		ret += indent + "}\n"
	} else {
		ret += "\n"
	}
	return ret
}

// FindFunction is a function type that receives a Section, its index in the
// parent's Section list, and the parent Section.
type FindFunction func(sec *Section, idx int, parent *Section) interface{}

// findFunc is a support function for FindFunc, that searches recursively for a
// section by name, and, if found, returns the section, its index in the
// parent's sections, and the parent. If no section by that name is found, the
// returned section is `nil`.
// If `recursive` is true, search also in sub-sections.
func findFunc(s *Section, name string, recursive bool) (*Section, int, *Section) {
	for idx, sec := range s.Sections {
		if sec.Name == name {
			return sec, idx, s
		}
	}
	// after searching in direct sub-sections, search recursively
	if recursive {
		for _, sec := range s.Sections {
			if found, idx, parent := findFunc(sec, name, true); found != nil {
				return found, idx, parent
			}
		}
	}
	return nil, -1, nil
}

// FindFunc searches for a sub-section with the given name, calls the
// specified FindFunction and returns its return value.
// If `recursive` is true, it will also search into subsections. If more
// than one section with the given name is found, only the first one is used.
func (s *Section) FindFunc(name string, recursive bool, f FindFunction) interface{} {
	found, idx, parent := findFunc(s, name, recursive)
	return f(found, idx, parent)
}

// Find searches for a sub-section with the given name. If `recursive` is
// true, it will also search into subsections. If more than one section with
// the given name is found, only the first one is returned.
// If no section is found, it returns `nil`.
func (s *Section) Find(name string, recursive bool) *Section {
	ret := s.FindFunc(name, recursive, func(s *Section, _ int, _ *Section) interface{} {
		return s
	})
	if sec, ok := ret.(*Section); ok {
		return sec
	}
	panic("not a section")
}

// Remove removes a sub-section from the current section. If `recursive` is
// true, it will also look into subsections. If more than one section with the
// given name is found, only the first one is removed.
// This function returns true if the section was found and removed, false
// otherwise.
func (s *Section) Remove(name string, recursive bool) bool {
	ret := s.FindFunc(name, recursive, func(sec *Section, idx int, parent *Section) interface{} {
		if sec == nil {
			return false
		}
		parent.Sections = append(parent.Sections[:idx], parent.Sections[idx+1:]...)
		return true
	})
	if removed, ok := ret.(bool); ok {
		return removed
	}
	panic("not a boolean")
}

// size returns the size in bytes of a section, taking the unit into account
func size(s *Section) int {
	switch s.Unit {
	case "k", "K":
		return s.Size * 1024
	case "m", "M":
		return s.Size * 1024 * 1024
	default:
		return s.Size
	}
}

func defrag(s *Section) bool {
	hasChanged := false
	start := 0
	for _, sec := range s.Sections {
		if sec.Start != nil && *sec.Start > start {
			log.Printf("Compacting section %s", sec.Name)
			// needs to be compacted
			hasChanged = true
			*sec.Start = start
		}
		start += size(sec)
		if yes := defrag(sec); yes {
			hasChanged = true
		}
	}
	return hasChanged
}

// Defrag defragments a flashmap so that no intermediate empty spaces are left.
// This function returns true if any change was made, false otherwise.
func (s *Section) Defrag() bool {
	return defrag(s)
}

// Parse parses a flashmap from an io.Reader and returns a Section object.
func Parse(fd io.Reader) (*Section, error) {
	parser, err := participle.Build(&Section{})
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(fd)
	if err != nil {
		return nil, err
	}
	flash := Section{}
	if err := parser.ParseString(string(data), &flash); err != nil {
		return nil, err
	}
	return &flash, nil
}
