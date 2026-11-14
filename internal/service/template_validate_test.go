package service

import "testing"

func TestValidateTemplateVariablesRejectsRetiredBuiltins(t *testing.T) {
	cases := []string{
		"你好 {{first_name}}，欢迎",
		"{{group_name}} 通知",
		"订单 {{ user_id }}",
	}
	for _, content := range cases {
		if err := validateTemplateVariables(content); err == nil {
			t.Fatalf("含退役变量的模板应被拒绝: %q", content)
		}
	}
}

func TestValidateTemplateVariablesAllowsDictionaryNames(t *testing.T) {
	cases := []string{
		"你好，{{问候语}}",
		"{{活动名}} 开始了",
		"没有变量也可以",
		"{{welcome}}",
	}
	for _, content := range cases {
		if err := validateTemplateVariables(content); err != nil {
			t.Fatalf("自定义词典变量应允许: %q -> %v", content, err)
		}
	}
}
