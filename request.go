package fauna

type QueryArgItem struct {
	Key   string
	Value any
}

func QueryArg(key string, value any) QueryArgItem {
	return QueryArgItem{
		Key:   key,
		Value: value,
	}
}

func QueryArguments(args ...QueryArgItem) map[string]interface{} {
	out := map[string]interface{}{}
	for n := range args {
		arg := args[n]
		out[arg.Key] = arg.Value
	}

	return out
}
