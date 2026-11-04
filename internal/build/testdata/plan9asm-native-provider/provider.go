package provider

var entry uintptr

//export native_component_export
func native_component_export() uintptr { return entry }

func Ordinary() uintptr { return entry }
