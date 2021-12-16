package templating

type JsonArrayBuilder struct {
	data []string
}

func NewJsonArrayBuilder() *JsonArrayBuilder {
	return &JsonArrayBuilder{
		data: []string{},
	}
}

func (j *JsonArrayBuilder) AddEntry(s string) {
	j.data = append(j.data, s)
}

func (j *JsonArrayBuilder) ToStringObj(itemname string) string {
	return "{" + j.ToStringElem(itemname) + "}"
}

func (j *JsonArrayBuilder) ToStringElem(itemname string) string {
	result := "\"" + itemname + "\": ["
	for index, entry := range j.data {
		if index != 0 {
			result += ","
		}
		result += entry
	}
	result += "]"
	return result
}
