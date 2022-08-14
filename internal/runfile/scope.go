package runfile

import "os"

// Assert captures an assertion for a runfile.
//
type Assert struct {
	Runfile string
	Line    int
	Test    string
	Message string
}

// VarExport captures a variable export.
//
type VarExport struct {
	VarName string
}

// AttrExport captures an attribute export.
//
type AttrExport struct {
	AttrName string
	VarName  string
}

// Scope isolates attrs, vars and exports
//
type Scope struct {
	Attrs       map[string]string // All keys uppercase. Keys include leading '.'
	Vars        map[string]string // Variables
	VarExports  []*VarExport      // Exported variables
	AttrExports []*AttrExport     // Exported attributes
	Asserts     []*Assert         // Assertions
}

// NewScope is a convenience method
//
func NewScope() *Scope {
	return &Scope{
		Attrs:       map[string]string{},
		Vars:        map[string]string{},
		VarExports:  []*VarExport{},
		AttrExports: []*AttrExport{},
		Asserts:     []*Assert{},
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

// ExportAttr adds an attribute name to the list of exports
//
func (s *Scope) ExportAttr(attrName string, varName string) {
	s.AttrExports = append(s.AttrExports, &AttrExport{AttrName: attrName, VarName: varName})
}

// GetAttrExports fetches the full list of attribute exports
//
func (s *Scope) GetAttrExports() []*AttrExport {
	return s.AttrExports
}

// GetVar fetches a variable
//
func (s *Scope) GetVar(key string) (string, bool) {
	val, ok := s.Vars[key]
	return val, ok
}

// PutVar sets a variable
//
func (s *Scope) PutVar(key, value string) {
	s.Vars[key] = value
}

// ExportVar adds a variable name to the list of exports
//
func (s *Scope) ExportVar(name string) {
	s.VarExports = append(s.VarExports, &VarExport{VarName: name})
}

// GetVarExports fetches the full list of variable exports
//
func (s *Scope) GetVarExports() []*VarExport {
	return s.VarExports
}

// AddAssert adds an assertion to the list of asserts
//
func (s *Scope) AddAssert(assert *Assert) {
	s.Asserts = append(s.Asserts, assert)
}
