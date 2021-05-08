package Utils

func GetIntInRange(min, max, value int) int {
	if value < min {
		value = min
	} else if value > max {
		value = max
	}
	return value
}

func GetOr(value interface{}, otherwise interface{}) interface{} {
	if value == nil {
		return otherwise
	}
	return value
}

func ValidateOrGet(value interface{}, getter func() interface{}) interface{} {
	if value == nil {
		return getter()
	}
	return value
}

func OnNullityCheck(value interface{}, onNull func(), onNotNull func()) {
	if value == nil {
		onNull()
	} else {
		onNotNull()
	}
}