// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package protocol

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"golang-refactoring.org/go-doctor/filesystem"
	"golang-refactoring.org/go-doctor/refactoring"
	"golang-refactoring.org/go-doctor/text"
)

type Command interface {
	Run(*State, map[string]interface{}) (Reply, error)
	Validate(*State, map[string]interface{}) (bool, error)
}

// -=-= About =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=

// TODO fix about text

type About struct {
	aboutText string
}

func (a *About) Run(state *State, input map[string]interface{}) (Reply, error) {
	if valid, err := a.Validate(state, input); valid {
		a.aboutText = "Go Doctor about text"
		return Reply{map[string]interface{}{"reply": "OK", "text": a.aboutText}}, nil
	} else {
		//err := errors.New("The about command requires a state of non-zero")
		return Reply{map[string]interface{}{"reply": "Error", "message": err.Error()}}, err
	}
}

func (a *About) Validate(state *State, input map[string]interface{}) (bool, error) {
	if state.State > 0 {
		return true, nil
	} else {
		return false, errors.New("The about command requires a state of non-zero")
	}
}

// -=-= List =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-

// TODO add in implementation of fileselection and textselection keys

type List struct {
	Fileselection []string       `json:"fileselection"`
	Textselection text.Selection `json:"textselection"`
	Quality       string         `json:"quality" chk:"in_testing|in_development|production"`
}

func (l *List) Run(state *State, input map[string]interface{}) (Reply, error) {

	if valid, err := l.Validate(state, input); valid {
		// get all of the refactoring names
		namesList := make([]map[string]string, 0)
		for shortName, refactoring := range refactoring.AllRefactorings() {
			namesList = append(namesList, map[string]string{"shortName": shortName, "name": refactoring.Description().Name})
		}
		return Reply{map[string]interface{}{"reply": "OK", "transformations": namesList}}, nil
	} else {
		return Reply{map[string]interface{}{"reply": "Error", "message": err.Error()}}, err
	}
}

func (l *List) Validate(state *State, input map[string]interface{}) (bool, error) {
	if state.State < 1 {
		err := errors.New("The about command requires a state of non-zero")
		return false, err
	}
	// check for required keys
	if _, found := input["quality"]; !found {
		err := errors.New("Quality key not found")
		return false, err
	} else {
		// check quality matches
		field, _ := reflect.TypeOf(l).Elem().FieldByName("Quality")
		qualityValidator := regexp.MustCompile(field.Tag.Get("chk"))

		if valid := qualityValidator.MatchString(input["quality"].(string)); !valid {
			return false, errors.New("Quality key must be \"in_testing|in_development|production\"")
		}
	}
	return true, nil
}

// -=-= Open =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-

// TODO open with version

type Open struct {
	Version float64 `json:"version"`
}

func (o *Open) Run(state *State, input map[string]interface{}) (Reply, error) {
	state.State = 1
	//printReply(Reply{"OK", ""})
	return Reply{map[string]interface{}{"reply": "OK"}}, nil
}

// basically useless until we implement versioning...
func (o *Open) Validate(state *State, input map[string]interface{}) (bool, error) {
	return true, nil
}

// -=-= Params =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-

type Params struct {
	Transformation string         `json:"transformation"`
	Fileselection  []string       `json:"fileselection"`
	Textselection  text.Selection `json:"textselection"`
}

func (p *Params) Run(state *State, input map[string]interface{}) (Reply, error) {
	//refactoring := refactoring.GetRefactoring("rename")
	if valid, err := p.Validate(state, input); valid {
		refactoring := refactoring.GetRefactoring(input["transformation"].(string))
		// since GetParams returns just a string, assume it as prompt and label
		params := make([]map[string]interface{}, 0)
		for _, param := range refactoring.Description().Params {
			params = append(params, map[string]interface{}{"label": param.Label, "prompt": param.Prompt, "type": reflect.TypeOf(param.DefaultValue), "default": param.DefaultValue})
		}
		return Reply{map[string]interface{}{"reply": "OK", "params": params}}, nil
	} else {
		return Reply{map[string]interface{}{"reply": "Error", "message": err.Error()}}, err
	}
}

func (p *Params) Validate(state *State, input map[string]interface{}) (bool, error) {
	if state.State < 2 {
		return false, errors.New("State of 2 (file system configured) is required")
	}
	if _, found := input["transformation"]; !found {
		return false, errors.New("Transformation key not found")
	}
	return true, nil
}

// -=-= Setdir =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-

type Setdir struct {
	Mode string `json:"mode" chk:"local|web"`
}

func (s *Setdir) Run(state *State, input map[string]interface{}) (Reply, error) {

	if valid, err := s.Validate(state, input); valid {
		// assuming everything is good?
		mode := input["mode"]
		state.Mode = mode.(string)

		// local mode? get directory and local filesystem
		if mode == "local" {
			state.Dir = input["directory"].(string)
			state.Filesystem = filesystem.NewLocalFileSystem()
		}

		// web mode? get that virtual filesystem
		if mode == "web" {
			//state.Filesystem = filesystem.NewVirtualFileSystem()
			return Reply{map[string]interface{}{"reply": "Error", "message": "Web mode not supported"}}, err
		}

		state.State = 2
		return Reply{map[string]interface{}{"reply": "OK"}}, nil
	} else {
		return Reply{map[string]interface{}{"reply": "Error", "message": err.Error()}}, err
	}

}

func (s *Setdir) Validate(state *State, input map[string]interface{}) (bool, error) {
	if state.State < 1 {
		return false, errors.New("State must be non-zero for \"setdir\" command")
	}

	// mode key?
	if mode, found := input["mode"]; !found {
		err := errors.New("\"mode\" key is required")
		return false, err
	} else {
		// validate the mode value
		field, _ := reflect.TypeOf(s).Elem().FieldByName("Mode")
		modeValidator := regexp.MustCompile(field.Tag.Get("chk"))
		if valid := modeValidator.MatchString(mode.(string)); !valid {
			return false, errors.New("\"mode\" key must be \"web|local\"")
		}
		// check for directory key if mode == local
		if mode == "local" {
			if _, found := input["directory"]; !found {
				return false, errors.New("\"directory\" key required if \"mode\" is local")
			}
			// validate directory
			fs := filesystem.NewLocalFileSystem()
			_, err := fs.ReadDir(input["directory"].(string))
			if err != nil {
				return false, err
			}
		}
	}
	return true, nil
}

// -=-= Xrun =-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-=-

type XRun struct {
	Transformation string                 `json:"transformation"`
	Fileselection  []string               `json:"fileselection"`
	Textselection  map[string]interface{} `json:"textselection"`
	Arguments      []interface{}          `json:"arguments"`
	Limit          int                    `json:"limit"`
	Mode           string                 `json:"mode" chk:"text|patch"`
}

// TODO implement
func (x *XRun) Run(state *State, input map[string]interface{}) (Reply, error) {
	if valid, err := x.Validate(state, input); !valid {
		return Reply{map[string]interface{}{"reply": "Error", "message": err.Error()}}, err
	}
	// setup TextSelection
	textselection := input["textselection"].(map[string]interface{})
	ts := &text.Selection{
		Filename:  filepath.Join(state.Dir, textselection["filename"].(string)),
		StartLine: int(textselection["startline"].(float64)),
		StartCol:  int(textselection["startcol"].(float64)),
		EndLine:   int(textselection["endline"].(float64)),
		EndCol:    int(textselection["endcol"].(float64)),
	}

	// get refactoring
	refac := refactoring.GetRefactoring(input["transformation"].(string))

	config := &refactoring.Config{
		FileSystem: state.Filesystem,
		Scope:      nil,
		Selection:  ts,
		Args:       input["arguments"].([]interface{}),
	}

	// run
	result := refac.Run(config)

	// grab logs
	logs := make([]map[string]interface{}, 0)
	for _, entry := range result.Log.Entries {
		var severity string
		switch entry.Severity {
		case refactoring.Info:
			// No prefix
		case refactoring.Warning:
			severity = "warning"
		case refactoring.Error:
			severity = "error"
		}
		log := map[string]interface{}{"severity": severity, "message": entry.Message}
		logs = append(logs, log)
	}

	changes := make([]map[string]string, 0)

	// if mode == patch or no mode was given
	if mode, found := input["mode"]; !found || mode.(string) == "patch" {
		for f, e := range result.Edits {
			var p *text.Patch
			var err error
			p, err = text.CreatePatchForFile(e, f)
			if err != nil {
				return Reply{map[string]interface{}{"reply": "Error", "message": err.Error()}}, err
			}
			diffFile, err := os.Create(strings.Join([]string{f, ".diff"}, ""))
			p.Write(f, f, diffFile)
			//fmt.Println(f)
			//fmt.Println(diffFile.Name())
			changes = append(changes, map[string]string{"filename": f, "patchFile": diffFile.Name()})
			diffFile.Close()
		}
	} else {
		for f, e := range result.Edits {
			content, err := text.ApplyToFile(e, f)
			if err != nil {
				return Reply{map[string]interface{}{"reply": "Error", "message": err.Error()}}, err
			}
			changes = append(changes, map[string]string{"filename": f, "content": string(content)})
		}
	}

	// filesystem changes
	var fschanges []map[string]string
	if len(result.FSChanges) > 0 {
		fschanges = make([]map[string]string, len(result.FSChanges))
		for i, change := range result.FSChanges {
			switch change := change.(type) {
			case *filesystem.CreateFile:
				fschanges[i] = map[string]string{"change": "create", "file": change.Path, "content": change.Contents}
			case *filesystem.Remove:
				fschanges[i] = map[string]string{"change": "delete", "path": change.Path}
			case *filesystem.Rename:
				fschanges[i] = map[string]string{"change": "rename", "from": change.Path, "to": change.NewName}
			}
		}
		// return with filesystem changes
		return Reply{map[string]interface{}{"reply": "OK", "description": refac.Description().Name, "log": logs, "files": changes, "fsChanges": fschanges}}, nil
	}

	// return without filesystem changes
	return Reply{map[string]interface{}{"reply": "OK", "description": refac.Description().Name, "log": logs, "files": changes}}, nil
}

// TODO validate TextSelection, FileSelection, arguments
func (x *XRun) Validate(state *State, input map[string]interface{}) (bool, error) {
	if state.State < 2 {
		return false, errors.New("State of 2 (file system configured) is required")
	}

	// check transformation is valid
	var valid bool
	for shortName, _ := range refactoring.AllRefactorings() {
		if shortName == input["transformation"].(string) {
			valid = true
		}
	}
	if !valid {
		return false, errors.New("Transformation given is not a valid refactoring name")
	}

	// check limit is > 0 if exists
	if limit, found := input["limit"]; found {
		if limit.(int) < 0 {
			return false, errors.New("\"limit\" key must be a positive integer")
		}
	}

	// check mode key if exists
	if mode, found := input["mode"]; found {
		field, _ := reflect.TypeOf(x).Elem().FieldByName("Mode")
		qualityValidator := regexp.MustCompile(field.Tag.Get("chk"))

		if valid := qualityValidator.MatchString(mode.(string)); !valid {
			return false, errors.New("\"mode\" key must be \"text|patch\"")
		}
	}

	// all good?
	return true, nil
}
