package service

import (
	"strings"
	"telegraph_uploader_v2/internal/database"
	"testing"
)

func TestApplyVariables(t *testing.T) {
	variables := []database.TitleVariable{
		{Key: "Author", Value: "John Doe"},
		{Key: "Date", Value: "2023-10-27"},
		{Key: "Chapter", Value: "105"},
		{Key: "Series", Value: "One Piece"},
		{Key: "Genre", Value: "Action"},
		{Key: "Empty", Value: ""},           // Should still replace
		{Key: "", Value: "ShouldNotHappen"}, // Should be ignored
	}
	content := "Title: {{Series}} Chapter {{Chapter}}\nAuthor: {{Author}}\nDate: {{Date}}\nGenre: {{Genre}}\nEmpty: {{Empty}}"
	expected := "Title: One Piece Chapter 105\nAuthor: John Doe\nDate: 2023-10-27\nGenre: Action\nEmpty: "

	result := applyVariables(content, variables)
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}

	// Test no variables
	result = applyVariables(content, nil)
	if result != content {
		t.Errorf("expected no change, got:\n%s", result)
	}

	// Test empty key in variables (should be ignored)
	if applyVariables("{{}}", []database.TitleVariable{{Key: "", Value: "Val"}}) != "{{}}" {
		t.Errorf("expected {{}} to remain {{}}")
	}
}

func BenchmarkApplyVariables(b *testing.B) {
	variables := []database.TitleVariable{
		{Key: "Author", Value: "John Doe"},
		{Key: "Date", Value: "2023-10-27"},
		{Key: "Chapter", Value: "105"},
		{Key: "Series", Value: "One Piece"},
		{Key: "Genre", Value: "Action"},
		{Key: "Var1", Value: "Val1"},
		{Key: "Var2", Value: "Val2"},
		{Key: "Var3", Value: "Val3"},
		{Key: "Var4", Value: "Val4"},
		{Key: "Var5", Value: "Val5"},
	}
	content := strings.Repeat("Title: {{Series}} Chapter {{Chapter}}\nAuthor: {{Author}}\nDate: {{Date}}\nGenre: {{Genre}}\nThis is a test for {{Series}} chapter {{Chapter}}.\n", 20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		applyVariables(content, variables)
	}
}
