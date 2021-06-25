package utils

func GetOr(obj interface{}, otherwise func() interface{}) interface{} {
	if obj != nil {
		return obj
	}
	return otherwise()
}

func ConditionalPick(cond bool, onTrue interface{}, onFalse interface{}) interface{} {
	if cond {
		return onTrue
	} else {
		return onFalse
	}
}

func ConditionalGet(cond bool, onTrue func() interface{}, onFalse func() interface{}) interface{} {
	if cond {
		return onTrue()
	} else {
		return onFalse()
	}
}

func SliceToSet(slice []interface{}) map[interface{}]bool {
	m := make(map[interface{}]bool)
	for _, v := range slice {
		m[v] = true
	}
	return m
}

func CopySet(set map[interface{}]bool) map[interface{}]bool {
	c := make(map[interface{}]bool)
	for k, v := range set {
		c[k] = v
	}
	return c
}

func SetIntersections(l map[interface{}]bool, r map[interface{}]bool) map[interface{}]bool {
	lCopy := CopySet(l)
	rCopy := CopySet(r)
	for k := range lCopy {
		if rCopy[k] {
			lCopy[k] = false
			rCopy[k] = false
		} else {
			rCopy[k] = true
		}
	}
	return rCopy
}

func StringArrayToInterfaceArray(arr []string) []interface{} {
	res := make([]interface{}, len(arr))
	for i := range arr {
		res[i] = arr[i]
	}
	return res
}
