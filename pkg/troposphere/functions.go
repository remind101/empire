package troposphere

// Ref provides a helper for the Ref function.
func Ref(ref interface{}) interface{} {
	switch v := ref.(type) {
	case NamedResource:
		ref = v.Name
	}
	return map[string]interface{}{"Ref": ref}
}

// GetAtt provides a helper for the GetAtt function.
func GetAtt(ref, attr interface{}) interface{} {
	switch v := ref.(type) {
	case NamedResource:
		ref = v.Name
	}
	return map[string][]interface{}{"Fn::GetAtt": []interface{}{ref, attr}}
}

// Equals is a helper for the Fn::Equals function.
func Equals(thing, value interface{}) interface{} {
	return map[string][]interface{}{"Fn::Equals": []interface{}{thing, value}}
}

// Join is a helper for the Fn::Join function.
func Join(delimiter string, things ...interface{}) interface{} {
	return map[string][]interface{}{"Fn::Join": []interface{}{delimiter, things}}
}
