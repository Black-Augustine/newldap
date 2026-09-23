package people

import "testing"

// TestSurname 姓氏推导：单字姓取第一字，复姓取前两字，英文名取末单词。
func TestSurname(t *testing.T) {
	cases := map[string]string{
		"张伟":   "张",
		"李娜":   "李",
		"王":    "王",
		"欧阳娜娜": "欧阳",
		"司马光":  "司马",
		"诸葛孔明": "诸葛",
		"John Smith": "Smith",
		"admin": "admin",
		"":     "",
	}
	for cn, want := range cases {
		if got := Surname(cn); got != want {
			t.Errorf("Surname(%q) = %q, want %q", cn, got, want)
		}
	}
}
