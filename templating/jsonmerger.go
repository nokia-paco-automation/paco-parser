package templating

import (
	jsonpatch "github.com/evanphx/json-patch"
	log "github.com/sirupsen/logrus"
)

type JsonMerger struct {
	data []byte
}

func NewJsonMerger() *JsonMerger {
	return &JsonMerger{
		data: []byte("{}"),
	}
}
func (j *JsonMerger) ToString() string {
	return string(j.data)
}
func (j *JsonMerger) Merge(a []byte) {
	result, err := jsonpatch.MergePatch(j.data, a)
	if err != nil {
		log.Fatalf("%v\n%s", err, string(a))
	}
	j.data = result
}
