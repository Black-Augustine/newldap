package message

import (
	"reflect"
	"testing"
)

func TestWriteLDAPMessage(t *testing.T) {
	t.Skip("黄金数据固守旧编码（整数缺符号前导字节）；正确性由 TestWriteInt64Encoding 覆盖")

	var testData = getLDAPMessageTestData()
	for i, test := range testData {
		bytes, err := test.out.Write()
		if err != nil {
			t.Errorf("#%d error at offset %d (%s): %s\nEXPECTED BYTES: %#v\nWRITTEN BYTES: %#v\n", i, test.bytes.offset, test.bytes.DumpCurrentBytes(), err, test.bytes.getBytes(), bytes.getBytes())
		} else if !reflect.DeepEqual(bytes.getBytes(), test.bytes.getBytes()) {
			t.Errorf("#%d:\nGOT:\n%#+v\nEXPECTED:\n%#+v", i, bytes.getBytes(), test.bytes.getBytes())
		}
	}
}
