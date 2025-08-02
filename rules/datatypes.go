// Package rules provides validation functionality.
package rules

// DataType defines the type that the input values are expected to be.
type DataType string

// DataType constants
const (
	DataTypeInt      DataType = "int"
	DataTypeFloat    DataType = "float"
	DataTypeString   DataType = "string"
	DataTypeBool     DataType = "bool"
	DataTypeTime     DataType = "time"
	DataTypeDuration DataType = "duration"
	DataTypeBytes    DataType = "bytes"
	DataTypeRegex    DataType = "regex"
)
