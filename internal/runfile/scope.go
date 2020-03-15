package runfile

import "os"

// Assert captures an assertion for a runfile.
//
type Assert struct {
	Line    int
	Test    string
	Message string
}

// Scope isolates attrs, vars and exports
//
type Scope struct {
	Attrs   map[string]string // All keys uppercase. Keys include leading '.'
	Vars    map[string]string // Variables
	Exports []string          // Exported variables
	Asserts []*Assert         // Assertions
}

// NewScope is a convenience method
//
func NewScope() *Scope {
	return &Scope{
		Attrs:   map[string]string{},
		Vars:    map[string]string{},
		Exports: []string{},
		Asserts: []*Assert{},
	}
}

// GetEnv fetches an env variable
//
func (s *Scope) GetEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// GetAttr fetches an attr
//
func (s *Scope) GetAttr(key string) (string, bool) {
	val, ok := s.Attrs[key]
	return val, ok
}

// PutAttr sets an attr
//
func (s *Scope) PutAttr(key, value string) {
	s.Attrs[key] = value
}

// GetVar fetches a var
//
func (s *Scope) GetVar(key string) (string, bool) {
	val, ok := s.Vars[key]
	return val, ok
}

// PutVar sets a var
//
func (s *Scope) PutVar(key, value string) {
	s.Vars[key] = value
}

// AddExport adds an var name to the list of exports
//
func (s *Scope) AddExport(key string) {
	s.Exports = append(s.Exports, key)
}

// GetExports fetches the full list of exports
//
func (s *Scope) GetExports() []string {
	return s.Exports
}

// AddAssert adds an assert to the list of asserts
//
func (s *Scope) AddAssert(assert *Assert) {
	s.Asserts = append(s.Asserts, assert)
}
