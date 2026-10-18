package ptr

func CopyBoolPtr(v *bool) *bool {
	c := false
	if v == nil {
		return &c
	}
	c = *v
	return &c
}

func MakeBool(v bool) *bool {
	return &v
}

func MakeUint(v uint) *uint {
	return &v
}

func MakeInt(v int) *int {
	return &v
}

func MakeString(v string) *string {
	return &v
}

func MakeInt64(v int64) *int64 {
	return &v
}
