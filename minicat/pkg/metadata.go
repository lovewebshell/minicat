package pkg

import (
	"reflect"
)

type MetadataType string

const (
	UnknownMetadataType        MetadataType = "UnknownMetadata"
	ApkMetadataType            MetadataType = "ApkMetadata"
	AlpmMetadataType           MetadataType = "AlpmMetadata"
	DpkgMetadataType           MetadataType = "DpkgMetadata"
	GemMetadataType            MetadataType = "GemMetadata"
	JavaMetadataType           MetadataType = "JavaMetadata"
	NpmPackageJSONMetadataType MetadataType = "NpmPackageJsonMetadata"
	RpmMetadataType            MetadataType = "RpmMetadata"
	PythonPackageMetadataType  MetadataType = "PythonPackageMetadata"
	KbPackageMetadataType      MetadataType = "KbPackageMetadata"
	GolangBinMetadataType      MetadataType = "GolangBinMetadata"
)

var AllMetadataTypes = []MetadataType{
	ApkMetadataType,
	AlpmMetadataType,
	DpkgMetadataType,
	GemMetadataType,
	JavaMetadataType,
	NpmPackageJSONMetadataType,
	RpmMetadataType,
	PythonPackageMetadataType,
	KbPackageMetadataType,
	GolangBinMetadataType,
}

var MetadataTypeByName = map[MetadataType]reflect.Type{
	ApkMetadataType:            reflect.TypeOf(ApkMetadata{}),
	AlpmMetadataType:           reflect.TypeOf(AlpmMetadata{}),
	JavaMetadataType:           reflect.TypeOf(JavaMetadata{}),
	NpmPackageJSONMetadataType: reflect.TypeOf(NpmPackageJSONMetadata{}),
	RpmMetadataType:            reflect.TypeOf(RpmMetadata{}),
	PythonPackageMetadataType:  reflect.TypeOf(PythonPackageMetadata{}),
	KbPackageMetadataType:      reflect.TypeOf(KbPackageMetadata{}),
	GolangBinMetadataType:      reflect.TypeOf(GolangBinMetadata{}),
}
