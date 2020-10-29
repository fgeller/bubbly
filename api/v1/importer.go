package v1

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/hcl/v2"
	"github.com/verifa/bubbly/api/core"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

// Compiler check to see that v1.Importer implements the Importer interface
var _ core.Importer = (*Importer)(nil)

// Importer represents an importer type
type Importer struct {
	*core.ResourceBlock

	Spec importerSpec `json:"spec"`
}

// NewImporter returns a new Importer
func NewImporter(resBlock *core.ResourceBlock) *Importer {
	return &Importer{
		ResourceBlock: resBlock,
	}
}

// Apply returns the output from applying a resource
func (i *Importer) Apply(ctx *core.ResourceContext) core.ResourceOutput {
	if err := i.decode(ctx.DecodeBody); err != nil {
		return core.ResourceOutput{
			Status: core.ResourceOutputFailure,
			Error:  fmt.Errorf("Failed to decode resource %s: %s", i.String(), err.Error()),
		}
	}

	if i == nil {
		return core.ResourceOutput{
			Status: core.ResourceOutputFailure,
			Error:  errors.New("Cannot get output of a null importer"),
			Value:  cty.NilVal,
		}
	}

	if i.Spec.Source == nil {
		return core.ResourceOutput{
			Status: core.ResourceOutputFailure,
			Error:  errors.New("Cannot get output of an importer with null source"),
			Value:  cty.NilVal,
		}
	}

	val, err := i.Spec.Source.Resolve()
	if err != nil {
		return core.ResourceOutput{
			Status: core.ResourceOutputFailure,
			Error:  fmt.Errorf("Failed to resolve importer source: %s", err.Error()),
			Value:  cty.NilVal,
		}
	}

	return core.ResourceOutput{
		Status: core.ResourceOutputSuccess,
		Error:  nil,
		Value:  val,
	}
}

func (i *Importer) SpecValue() core.ResourceSpec {
	return &i.Spec
}

// decode is responsible for decoding any necessary hcl.Body inside Importer
func (i *Importer) decode(decode core.DecodeBodyFn) error {
	// decode the resource spec into the importer's Spec
	if err := decode(i, i.SpecHCL.Body, &i.Spec); err != nil {
		return fmt.Errorf(`Failed to decode "%s" body spec: %s`, i.String(), err.Error())
	}

	// based on the type of the importer, initiate the importer's Source
	switch i.Spec.Type {
	case jsonImporterType:
		i.Spec.Source = &jsonSource{}
	case xmlImporterType:
		i.Spec.Source = &xmlSource{}
	default:
		panic(fmt.Sprintf("Unsupported importer resource type %s", i.Spec.Type))
	}

	// decode the source HCL into the importer's Source
	if err := decode(i, i.Spec.SourceHCL.Body, i.Spec.Source); err != nil {
		return fmt.Errorf(`Failed to decode importer source: %s`, err.Error())
	}

	return nil
}

var _ core.ResourceSpec = (*importerSpec)(nil)

// importerSpec defines the spec for an importer
type importerSpec struct {
	Inputs InputDeclarations `hcl:"input,block"`
	// the type is either json, xml, rest, etc.
	Type      importerType `hcl:"type,attr"`
	SourceHCL struct {
		Body hcl.Body `hcl:",remain"`
	} `hcl:"source,block"`
	// Source stores the actual value for SourceHCL
	Source source
}

// importerType defines the type of an importer
type importerType string

const (
	jsonImporterType importerType = "json"
	xmlImporterType               = "xml"
)

// source is an interface for the different data sources that an Importer
// can have
type source interface {
	// returns an interface{} containing the parsed XML, JSON data, that should
	// be converted into the Output cty.Value
	Resolve() (cty.Value, error)
}

var _ source = (*jsonSource)(nil)

// jsonSource represents the importer type for using a JSON file as the input
type jsonSource struct {
	File string `hcl:"file,attr"`
	// the format of the raw input data defined as a cty.Type
	Format cty.Type `hcl:"format,attr"`
}

// Resolve returns a cty.Value representation of the parsed JSON file
func (s *jsonSource) Resolve() (cty.Value, error) {

	var barr []byte
	var err error

	// FIXME reading the whole file at once may be too much
	barr, err = ioutil.ReadFile(s.File)
	if err != nil {
		return cty.NilVal, err
	}

	// Attempt to unmarshall the data into an empty interface data type
	var data interface{}
	err = json.Unmarshal(barr, &data)
	if err != nil {
		return cty.NilVal, err
	}

	val, err := gocty.ToCtyValue(data, s.Format)
	if err != nil {
		return cty.NilVal, nil
	}

	return val, nil
}

var _ source = (*xmlSource)(nil)

// xmlSource represents the importer type for using an XML file as the input
type xmlSource struct {
	File string `hcl:"file,attr"`
	// the format of the raw input data defined as a cty.Type
	Format cty.Type `hcl:"format,attr"`
}

// Resolve returns a cty.Value representation of the XML file
func (s *xmlSource) Resolve() (cty.Value, error) {
	return cty.NilVal, errors.New("not implemented")
}
